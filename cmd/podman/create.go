package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/containers/storage"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/signal"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/pkg/errors"
	"github.com/projectatomic/libpod/cmd/podman/libpodruntime"
	"github.com/projectatomic/libpod/libpod"
	"github.com/projectatomic/libpod/libpod/image"
	"github.com/projectatomic/libpod/pkg/inspect"
	cc "github.com/projectatomic/libpod/pkg/spec"
	"github.com/projectatomic/libpod/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	defaultEnvVariables = map[string]string{
		"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"TERM": "xterm",
	}
)

var createDescription = "Creates a new container from the given image or" +
	" storage and prepares it for running the specified command. The" +
	" container ID is then printed to stdout. You can then start it at" +
	" any time with the podman start <container_id> command. The container" +
	" will be created with the initial state 'created'."

var createCommand = cli.Command{
	Name:                   "create",
	Usage:                  "create but do not start a container",
	Description:            createDescription,
	Flags:                  createFlags,
	Action:                 createCmd,
	ArgsUsage:              "IMAGE [COMMAND [ARG...]]",
	SkipArgReorder:         true,
	UseShortOptionHandling: true,
}

func createCmd(c *cli.Context) error {
	// TODO should allow user to create based off a directory on the host not just image
	// Need CLI support for this

	if err := validateFlags(c, createFlags); err != nil {
		return err
	}

	if c.String("cidfile") != "" {
		if _, err := os.Stat(c.String("cidfile")); err == nil {
			return errors.Errorf("container id file exists. ensure another container is not using it or delete %s", c.String("cidfile"))
		}
		if err := libpod.WriteFile("", c.String("cidfile")); err != nil {
			return errors.Wrapf(err, "unable to write cidfile %s", c.String("cidfile"))
		}
	}

	if len(c.Args()) < 1 {
		return errors.Errorf("image name or ID is required")
	}

	mappings, err := util.ParseIDMapping(c.StringSlice("uidmap"), c.StringSlice("gidmap"), c.String("subuidmap"), c.String("subgidmap"))
	if err != nil {
		return err
	}
	storageOpts := storage.DefaultStoreOptions
	storageOpts.UIDMap = mappings.UIDMap
	storageOpts.GIDMap = mappings.GIDMap

	runtime, err := libpodruntime.GetRuntimeWithStorageOpts(c, &storageOpts)
	if err != nil {
		return errors.Wrapf(err, "error creating libpod runtime")
	}
	defer runtime.Shutdown(false)

	rtc := runtime.GetConfig()
	ctx := getContext()

	newImage, err := runtime.ImageRuntime().New(ctx, c.Args()[0], rtc.SignaturePolicyPath, "", os.Stderr, nil, image.SigningOptions{}, false, false)
	if err != nil {
		return err
	}
	data, err := newImage.Inspect(ctx)
	createConfig, err := parseCreateOpts(ctx, c, runtime, newImage.Names()[0], data)
	if err != nil {
		return err
	}
	useImageVolumes := createConfig.ImageVolumeType == "bind"

	runtimeSpec, err := cc.CreateConfigToOCISpec(createConfig)
	if err != nil {
		return err
	}
	options, err := createConfig.GetContainerCreateOptions()
	if err != nil {
		return errors.Wrapf(err, "unable to parse new container options")
	}
	// Gather up the options for NewContainer which consist of With... funcs
	options = append(options, libpod.WithRootFSFromImage(createConfig.ImageID, createConfig.Image, useImageVolumes))
	options = append(options, libpod.WithSELinuxLabels(createConfig.ProcessLabel, createConfig.MountLabel))
	options = append(options, libpod.WithConmonPidFile(createConfig.ConmonPidFile))
	options = append(options, libpod.WithLabels(createConfig.Labels))
	options = append(options, libpod.WithUser(createConfig.User))
	options = append(options, libpod.WithShmDir(createConfig.ShmDir))
	options = append(options, libpod.WithShmSize(createConfig.Resources.ShmSize))
	options = append(options, libpod.WithGroups(createConfig.GroupAdd))
	options = append(options, libpod.WithIDMappings(*createConfig.IDMappings))
	ctr, err := runtime.NewContainer(ctx, runtimeSpec, options...)
	if err != nil {
		return err
	}

	createConfigJSON, err := json.Marshal(createConfig)
	if err != nil {
		return err
	}
	if err := ctr.AddArtifact("create-config", createConfigJSON); err != nil {
		return err
	}

	logrus.Debug("new container created ", ctr.ID())

	if c.String("cidfile") != "" {
		err := libpod.WriteFile(ctr.ID(), c.String("cidfile"))
		if err != nil {
			logrus.Error(err)
		}
	}
	fmt.Printf("%s\n", ctr.ID())
	return nil
}

func parseSecurityOpt(config *cc.CreateConfig, securityOpts []string) error {
	var (
		labelOpts []string
		err       error
	)

	if config.PidMode.IsHost() {
		labelOpts = append(labelOpts, label.DisableSecOpt()...)
	} else if config.PidMode.IsContainer() {
		ctr, err := config.Runtime.LookupContainer(config.PidMode.Container())
		if err != nil {
			return errors.Wrapf(err, "container %q not found", config.PidMode.Container())
		}
		labelOpts = append(labelOpts, label.DupSecOpt(ctr.ProcessLabel())...)
	}

	if config.IpcMode.IsHost() {
		labelOpts = append(labelOpts, label.DisableSecOpt()...)
	} else if config.IpcMode.IsContainer() {
		ctr, err := config.Runtime.LookupContainer(config.IpcMode.Container())
		if err != nil {
			return errors.Wrapf(err, "container %q not found", config.IpcMode.Container())
		}
		labelOpts = append(labelOpts, label.DupSecOpt(ctr.ProcessLabel())...)
	}

	for _, opt := range securityOpts {
		if opt == "no-new-privileges" {
			config.NoNewPrivs = true
		} else {
			con := strings.SplitN(opt, "=", 2)
			if len(con) != 2 {
				return fmt.Errorf("Invalid --security-opt 1: %q", opt)
			}

			switch con[0] {
			case "label":
				labelOpts = append(labelOpts, con[1])
			case "apparmor":
				config.ApparmorProfile = con[1]
			case "seccomp":
				config.SeccompProfilePath = con[1]
			default:
				return fmt.Errorf("Invalid --security-opt 2: %q", opt)
			}
		}
	}

	if config.SeccompProfilePath == "" {
		if _, err := os.Stat(libpod.SeccompOverridePath); err == nil {
			config.SeccompProfilePath = libpod.SeccompOverridePath
		} else {
			if !os.IsNotExist(err) {
				return errors.Wrapf(err, "can't check if %q exists", libpod.SeccompOverridePath)
			}
			if _, err := os.Stat(libpod.SeccompDefaultPath); err != nil {
				if !os.IsNotExist(err) {
					return errors.Wrapf(err, "can't check if %q exists", libpod.SeccompDefaultPath)
				}
			} else {
				config.SeccompProfilePath = libpod.SeccompDefaultPath
			}
		}
	}
	config.ProcessLabel, config.MountLabel, err = label.InitLabels(labelOpts)
	return err
}

// isPortInPortBindings determines if an exposed host port is in user
// provided ports
func isPortInPortBindings(pb map[nat.Port][]nat.PortBinding, port nat.Port) bool {
	var hostPorts []string
	for _, i := range pb {
		hostPorts = append(hostPorts, i[0].HostPort)
	}
	return util.StringInSlice(port.Port(), hostPorts)
}

// isPortInImagePorts determines if an exposed host port was given to us by metadata
// in the image itself
func isPortInImagePorts(exposedPorts map[string]struct{}, port string) bool {
	for i := range exposedPorts {
		fields := strings.Split(i, "/")
		if port == fields[0] {
			return true
		}
	}
	return false
}

// Parses CLI options related to container creation into a config which can be
// parsed into an OCI runtime spec
func parseCreateOpts(ctx context.Context, c *cli.Context, runtime *libpod.Runtime, imageName string, data *inspect.ImageData) (*cc.CreateConfig, error) {
	var (
		inputCommand, command                                    []string
		memoryLimit, memoryReservation, memorySwap, memoryKernel int64
		blkioWeight                                              uint16
	)
	idmappings, err := util.ParseIDMapping(c.StringSlice("uidmap"), c.StringSlice("gidmap"), c.String("subuidname"), c.String("subgidname"))
	if err != nil {
		return nil, err
	}

	imageID := data.ID

	if len(c.Args()) > 1 {
		inputCommand = c.Args()[1:]
	}

	sysctl, err := validateSysctl(c.StringSlice("sysctl"))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid value for sysctl")
	}

	if c.String("memory") != "" {
		memoryLimit, err = units.RAMInBytes(c.String("memory"))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for memory")
		}
	}
	if c.String("memory-reservation") != "" {
		memoryReservation, err = units.RAMInBytes(c.String("memory-reservation"))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for memory-reservation")
		}
	}
	if c.String("memory-swap") != "" {
		memorySwap, err = units.RAMInBytes(c.String("memory-swap"))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for memory-swap")
		}
	}
	if c.String("kernel-memory") != "" {
		memoryKernel, err = units.RAMInBytes(c.String("kernel-memory"))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for kernel-memory")
		}
	}
	if c.String("blkio-weight") != "" {
		u, err := strconv.ParseUint(c.String("blkio-weight"), 10, 16)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for blkio-weight")
		}
		blkioWeight = uint16(u)
	}

	if err = parseVolumes(c.StringSlice("volume")); err != nil {
		return nil, err
	}

	tty := c.Bool("tty")

	pidMode := container.PidMode(c.String("pid"))
	if !pidMode.Valid() {
		return nil, errors.Errorf("--pid %q is not valid", c.String("pid"))
	}

	usernsMode := container.UsernsMode(c.String("userns"))
	if !usernsMode.Valid() {
		return nil, errors.Errorf("--userns %q is not valid", c.String("userns"))
	}

	if c.Bool("detach") && c.Bool("rm") {
		return nil, errors.Errorf("--rm and --detach can not be specified together")
	}
	if c.Int64("cpu-period") != 0 && c.Float64("cpus") > 0 {
		return nil, errors.Errorf("--cpu-period and --cpus cannot be set together")
	}
	if c.Int64("cpu-quota") != 0 && c.Float64("cpus") > 0 {
		return nil, errors.Errorf("--cpu-quota and --cpus cannot be set together")
	}

	utsMode := container.UTSMode(c.String("uts"))
	if !utsMode.Valid() {
		return nil, errors.Errorf("--uts %q is not valid", c.String("uts"))
	}
	ipcMode := container.IpcMode(c.String("ipc"))
	if !ipcMode.Valid() {
		return nil, errors.Errorf("--ipc %q is not valid", ipcMode)
	}
	shmDir := ""
	if ipcMode.IsHost() {
		shmDir = "/dev/shm"
	} else if ipcMode.IsContainer() {
		ctr, err := runtime.LookupContainer(ipcMode.Container())
		if err != nil {
			return nil, errors.Wrapf(err, "container %q not found", ipcMode.Container())
		}
		shmDir = ctr.ShmDir()
	}

	// USER
	user := c.String("user")
	if user == "" {
		user = data.ContainerConfig.User
	}

	// STOP SIGNAL
	stopSignal := syscall.SIGTERM
	signalString := data.ContainerConfig.StopSignal
	if c.IsSet("stop-signal") {
		signalString = c.String("stop-signal")
	}
	if signalString != "" {
		stopSignal, err = signal.ParseSignal(signalString)
		if err != nil {
			return nil, err
		}
	}

	// ENVIRONMENT VARIABLES
	env := defaultEnvVariables
	for _, e := range data.ContainerConfig.Env {
		split := strings.SplitN(e, "=", 2)
		if len(split) > 1 {
			env[split[0]] = split[1]
		} else {
			env[split[0]] = ""
		}
	}
	if err := readKVStrings(env, c.StringSlice("env-file"), c.StringSlice("env")); err != nil {
		return nil, errors.Wrapf(err, "unable to process environment variables")
	}

	// LABEL VARIABLES
	labels, err := getAllLabels(c.StringSlice("label-file"), c.StringSlice("label"))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to process labels")
	}
	for key, val := range data.ContainerConfig.Labels {
		if _, ok := labels[key]; !ok {
			labels[key] = val
		}
	}

	// WORKING DIRECTORY
	workDir := "/"
	if c.IsSet("workdir") {
		workDir = c.String("workdir")
	} else if data.ContainerConfig.WorkingDir != "" {
		workDir = data.ContainerConfig.WorkingDir
	}

	// ENTRYPOINT
	// User input entrypoint takes priority over image entrypoint
	entrypoint := c.StringSlice("entrypoint")
	if len(entrypoint) == 0 {
		entrypoint = data.ContainerConfig.Entrypoint
	}
	// if entrypoint=, we need to clear the entrypoint
	if len(entrypoint) == 1 && c.IsSet("entrypoint") && strings.Join(c.StringSlice("entrypoint"), "") == "" {
		entrypoint = []string{}
	}
	// Build the command
	// If we have an entry point, it goes first
	if len(entrypoint) > 0 {
		command = entrypoint
	}
	if len(inputCommand) > 0 {
		// User command overrides data CMD
		command = append(command, inputCommand...)
	} else if len(data.ContainerConfig.Cmd) > 0 && !c.IsSet("entrypoint") {
		// If not user command, add CMD
		command = append(command, data.ContainerConfig.Cmd...)
	}

	if len(command) == 0 {
		return nil, errors.Errorf("No command specified on command line or as CMD or ENTRYPOINT in this image")
	}

	// EXPOSED PORTS
	portBindings, err := cc.ExposedPorts(c.StringSlice("expose"), c.StringSlice("publish"), c.Bool("publish-all"), data.ContainerConfig.ExposedPorts)
	if err != nil {
		return nil, err
	}

	// SHM Size
	shmSize, err := units.FromHumanSize(c.String("shm-size"))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to translate --shm-size")
	}
	// Network
	// Both --network and --net have default values of 'bridge'
	// --net only overrides --network when --network is not explicitly
	// set and --net is.
	if c.IsSet("network") && c.IsSet("net") {
		return nil, errors.Errorf("cannot use --network and --net together.  use only --network instead")
	}
	networkMode := c.String("network")
	if !c.IsSet("network") && c.IsSet("net") {
		networkMode = c.String("net")
	}

	// Verify the additional hosts are in correct format
	for _, host := range c.StringSlice("add-host") {
		if _, err := validateExtraHost(host); err != nil {
			return nil, err
		}
	}

	// Check for . and dns-search domains
	if util.StringInSlice(".", c.StringSlice("dns-search")) && len(c.StringSlice("dns-search")) > 1 {
		return nil, errors.Errorf("cannot pass additional search domains when also specifying '.'")
	}

	// Validate domains are good
	for _, dom := range c.StringSlice("dns-search") {
		if _, err := validateDomain(dom); err != nil {
			return nil, err
		}
	}
	ImageVolumes := data.ContainerConfig.Volumes

	var imageVolType = map[string]string{
		"bind":   "",
		"tmpfs":  "",
		"ignore": "",
	}
	if _, ok := imageVolType[c.String("image-volume")]; !ok {
		return nil, errors.Errorf("invalid image-volume type %q. Pick one of bind, tmpfs, or ignore", c.String("image-volume"))
	}

	config := &cc.CreateConfig{
		Runtime:           runtime,
		BuiltinImgVolumes: ImageVolumes,
		ConmonPidFile:     c.String("conmon-pidfile"),
		ImageVolumeType:   c.String("image-volume"),
		CapAdd:            c.StringSlice("cap-add"),
		CapDrop:           c.StringSlice("cap-drop"),
		CgroupParent:      c.String("cgroup-parent"),
		Command:           command,
		Detach:            c.Bool("detach"),
		Devices:           c.StringSlice("device"),
		DNSOpt:            c.StringSlice("dns-opt"),
		DNSSearch:         c.StringSlice("dns-search"),
		DNSServers:        c.StringSlice("dns"),
		Entrypoint:        entrypoint,
		Env:               env,
		//ExposedPorts:   ports,
		GroupAdd:       c.StringSlice("group-add"),
		Hostname:       c.String("hostname"),
		HostAdd:        c.StringSlice("add-host"),
		IDMappings:     idmappings,
		Image:          imageName,
		ImageID:        imageID,
		Interactive:    c.Bool("interactive"),
		IP6Address:     c.String("ipv6"),
		IPAddress:      c.String("ip"),
		Labels:         labels,
		LinkLocalIP:    c.StringSlice("link-local-ip"),
		LogDriver:      c.String("log-driver"),
		LogDriverOpt:   c.StringSlice("log-opt"),
		MacAddress:     c.String("mac-address"),
		Name:           c.String("name"),
		Network:        networkMode,
		NetworkAlias:   c.StringSlice("network-alias"),
		IpcMode:        ipcMode,
		NetMode:        container.NetworkMode(networkMode),
		UtsMode:        utsMode,
		PidMode:        pidMode,
		Pod:            c.String("pod"),
		Privileged:     c.Bool("privileged"),
		Publish:        c.StringSlice("publish"),
		PublishAll:     c.Bool("publish-all"),
		PortBindings:   portBindings,
		Quiet:          c.Bool("quiet"),
		ReadOnlyRootfs: c.Bool("read-only"),
		Resources: cc.CreateResourceConfig{
			BlkioWeight:       blkioWeight,
			BlkioWeightDevice: c.StringSlice("blkio-weight-device"),
			CPUShares:         c.Uint64("cpu-shares"),
			CPUPeriod:         c.Uint64("cpu-period"),
			CPUsetCPUs:        c.String("cpuset-cpus"),
			CPUsetMems:        c.String("cpuset-mems"),
			CPUQuota:          c.Int64("cpu-quota"),
			CPURtPeriod:       c.Uint64("cpu-rt-period"),
			CPURtRuntime:      c.Int64("cpu-rt-runtime"),
			CPUs:              c.Float64("cpus"),
			DeviceReadBps:     c.StringSlice("device-read-bps"),
			DeviceReadIOps:    c.StringSlice("device-read-iops"),
			DeviceWriteBps:    c.StringSlice("device-write-bps"),
			DeviceWriteIOps:   c.StringSlice("device-write-iops"),
			DisableOomKiller:  c.Bool("oom-kill-disable"),
			ShmSize:           shmSize,
			Memory:            memoryLimit,
			MemoryReservation: memoryReservation,
			MemorySwap:        memorySwap,
			MemorySwappiness:  c.Int("memory-swappiness"),
			KernelMemory:      memoryKernel,
			OomScoreAdj:       c.Int("oom-score-adj"),

			PidsLimit: c.Int64("pids-limit"),
			Ulimit:    c.StringSlice("ulimit"),
		},
		Rm:          c.Bool("rm"),
		ShmDir:      shmDir,
		StopSignal:  stopSignal,
		StopTimeout: c.Uint("stop-timeout"),
		Sysctl:      sysctl,
		Tmpfs:       c.StringSlice("tmpfs"),
		Tty:         tty,
		User:        user,
		UsernsMode:  usernsMode,
		Volumes:     c.StringSlice("volume"),
		WorkDir:     workDir,
	}

	if !config.Privileged {
		if err := parseSecurityOpt(config, c.StringSlice("security-opt")); err != nil {
			return nil, err
		}
	}
	config.SecurityOpts = c.StringSlice("security-opt")
	warnings, err := verifyContainerResources(config, false)
	if err != nil {
		return nil, err
	}
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, warning)
	}
	return config, nil
}
