package serviceapi

import (
	"context"
	"fmt"
	goRuntime "runtime"
	"time"

	"github.com/containers/libpod/libpod"
	"github.com/containers/libpod/libpod/define"
	libpodImage "github.com/containers/libpod/libpod/image"
	libpodInspect "github.com/containers/libpod/pkg/inspect"
	docker "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerNetwork "github.com/docker/docker/api/types/network"
	"github.com/pkg/errors"
)

type ImageInspect struct {
	docker.ImageInspect
}

type ContainerConfig struct {
	dockerContainer.Config
}

type ImageSummary struct {
	docker.ImageSummary
}

type Info struct {
	docker.Info
	BuildahVersion string
	CgroupVersion  string
	Rootless       bool
	SwapFree       int64
	SwapTotal      int64
	Uptime         string
}

type Container struct {
	docker.ContainerJSON
	docker.ContainerCreateConfig
}

type ContainerStats struct {
	docker.ContainerStats
}

type Ping struct {
	docker.Ping
}

type Version struct {
	docker.Version
}

type DiskUsage struct {
	docker.DiskUsage
}

type VolumesPruneReport struct {
	docker.VolumesPruneReport
}

type ImagesPruneReport struct {
	docker.ImagesPruneReport
}

type BuildCachePruneReport struct {
	docker.BuildCachePruneReport
}

type NetworkPruneReport struct {
	docker.NetworksPruneReport
}

type ConfigCreateResponse struct {
	docker.ConfigCreateResponse
}

type PushResult struct {
	docker.PushResult
}

type BuildResult struct {
	docker.BuildResult
}

type ContainerWaitOKBody struct {
	StatusCode int
	Error      struct {
		Message string
	}
}

type CreateContainer struct {
	Name string
	dockerContainer.Config
	HostConfig       dockerContainer.HostConfig
	NetworkingConfig dockerNetwork.NetworkingConfig
}

func ImageToImageSummary(l *libpodImage.Image) (*ImageSummary, error) {
	containers, err := l.Containers()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain Containers for image %s", l.ID())
	}
	containerCount := len(containers)

	var digests []string
	for _, d := range l.Digests() {
		digests = append(digests, string(d))
	}

	tags, err := l.RepoTags()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain RepoTags for image %s", l.ID())
	}

	// FIXME: GetParent() panics
	// parent, err := l.GetParent(context.TODO())
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "Failed to obtain ParentID for image %s", l.ID())
	// }

	labels, err := l.Labels(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain Labels for image %s", l.ID())
	}

	size, err := l.Size(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain Size for image %s", l.ID())
	}
	return &ImageSummary{docker.ImageSummary{
		Containers:  int64(containerCount),
		Created:     l.Created().Unix(),
		ID:          l.ID(),
		Labels:      labels,
		ParentID:    "parent.ID()",
		RepoDigests: digests,
		RepoTags:    tags,
		SharedSize:  0,
		Size:        int64(*size),
		VirtualSize: int64(*size),
	}}, nil
}

func ImageDataToImageInspect(l *libpodInspect.ImageData) (*ImageInspect, error) {
	return &ImageInspect{docker.ImageInspect{
		Architecture:    l.Architecture,
		Author:          l.Author,
		Comment:         l.Comment,
		Config:          &dockerContainer.Config{},
		Container:       "",
		ContainerConfig: nil,
		Created:         l.Created.Format(time.RFC3339Nano),
		DockerVersion:   "",
		GraphDriver:     docker.GraphDriverData{},
		ID:              l.ID,
		Metadata:        docker.ImageMetadata{},
		Os:              l.Os,
		OsVersion:       l.Version,
		Parent:          l.Parent,
		RepoDigests:     l.RepoDigests,
		RepoTags:        l.RepoTags,
		RootFS:          docker.RootFS{},
		Size:            l.Size,
		Variant:         "",
		VirtualSize:     l.VirtualSize,
	}}, nil
}

func LibpodToContainer(l *libpod.Container, infoData []define.InfoData) (*Container, error) {

	hostInfo := infoData[0].Data
	// storeInfo := infoData[1].Data

	_, imageId := l.Image()

	sizeRW, err := l.RWSize()
	if err != nil {
		return nil, err
	}

	SizeRootFs, err := l.RootFsSize()
	if err != nil {
		return nil, err
	}

	bindMounts, err := l.BindMounts()
	if err != nil {
		return nil, err
	}

	mountPoints := make([]docker.MountPoint, len(bindMounts))
	i := 0
	for k, v := range bindMounts {
		mountPoints[i] = docker.MountPoint{
			Type:        "",
			Name:        "",
			Source:      k,
			Destination: v,
			Driver:      "",
			Mode:        "",
			RW:          false,
			Propagation: "",
		}
		i++
	}

	return &Container{docker.ContainerJSON{
		ContainerJSONBase: &docker.ContainerJSONBase{
			ID:              l.ID(),
			Created:         l.CreatedTime().Format(time.RFC3339),
			Path:            l.CheckpointPath(),
			Args:            l.Command(),
			State:           nil,
			Image:           imageId,
			ResolvConfPath:  "",
			HostnamePath:    "/etc/hostname",
			HostsPath:       "/etc/hosts",
			LogPath:         l.LogPath(),
			Node:            nil,
			Name:            l.Name(),
			RestartCount:    int(l.RestartRetries()),
			Driver:          "",
			Platform:        fmt.Sprintf("%s/%s/%s", goRuntime.GOOS, goRuntime.GOARCH, hostInfo["Distribution"].(map[string]interface{})["distribution"].(string)),
			MountLabel:      l.MountLabel(),
			ProcessLabel:    l.ProcessLabel(),
			AppArmorProfile: "",
			ExecIDs:         nil,
			HostConfig:      nil,
			GraphDriver:     docker.GraphDriverData{},
			SizeRw:          &sizeRW,
			SizeRootFs:      &SizeRootFs,
		},
		Mounts:          mountPoints,
		Config:          nil,
		NetworkSettings: nil,
	},
		docker.ContainerCreateConfig{},
	}, nil
}
