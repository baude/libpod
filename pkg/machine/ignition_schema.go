package machine

/*
	This file was taken from https://github.com/coreos/ignition/blob/master/config/v3_2/types/schema.go in an effort to
	use more of the core-os structs but not fully commit to bringing their api in.

	// generated by "schematyper --package=types config/v3_2/schema/ignition.json -o config/v3_2/types/ignition_schema.go --root-type=Config" -- DO NOT EDIT
*/

type Clevis struct {
	Custom    *Custom `json:"custom,omitempty"`
	Tang      []Tang  `json:"tang,omitempty"`
	Threshold *int    `json:"threshold,omitempty"`
	Tpm2      *bool   `json:"tpm2,omitempty"`
}

type Config struct {
	Ignition Ignition `json:"ignition"`
	Passwd   Passwd   `json:"passwd,omitempty"`
	Storage  Storage  `json:"storage,omitempty"`
	Systemd  Systemd  `json:"systemd,omitempty"`
}

type Custom struct {
	Config       string `json:"config"`
	NeedsNetwork *bool  `json:"needsNetwork,omitempty"`
	Pin          string `json:"pin"`
}

type Device string

type Directory struct {
	Node
	DirectoryEmbedded1
}

type DirectoryEmbedded1 struct {
	Mode *int `json:"mode,omitempty"`
}

type Disk struct {
	Device     string      `json:"device"`
	Partitions []Partition `json:"partitions,omitempty"`
	WipeTable  *bool       `json:"wipeTable,omitempty"`
}

type Dropin struct {
	Contents *string `json:"contents,omitempty"`
	Name     string  `json:"name"`
}

type File struct {
	Node
	FileEmbedded1
}

type FileEmbedded1 struct {
	Append   []Resource `json:"append,omitempty"`
	Contents Resource   `json:"contents,omitempty"`
	Mode     *int       `json:"mode,omitempty"`
}

type Filesystem struct {
	Device         string             `json:"device"`
	Format         *string            `json:"format,omitempty"`
	Label          *string            `json:"label,omitempty"`
	MountOptions   []MountOption      `json:"mountOptions,omitempty"`
	Options        []FilesystemOption `json:"options,omitempty"`
	Path           *string            `json:"path,omitempty"`
	UUID           *string            `json:"uuid,omitempty"`
	WipeFilesystem *bool              `json:"wipeFilesystem,omitempty"`
}

type FilesystemOption string

type Group string

type HTTPHeader struct {
	Name  string  `json:"name"`
	Value *string `json:"value,omitempty"`
}

type HTTPHeaders []HTTPHeader

type Ignition struct {
	Config   IgnitionConfig `json:"config,omitempty"`
	Proxy    Proxy          `json:"proxy,omitempty"`
	Security Security       `json:"security,omitempty"`
	Timeouts Timeouts       `json:"timeouts,omitempty"`
	Version  string         `json:"version,omitempty"`
}

type IgnitionConfig struct {
	Merge   []Resource `json:"merge,omitempty"`
	Replace Resource   `json:"replace,omitempty"`
}

type Link struct {
	Node
	LinkEmbedded1
}

type LinkEmbedded1 struct {
	Hard   *bool  `json:"hard,omitempty"`
	Target string `json:"target"`
}

type Luks struct {
	Clevis     *Clevis      `json:"clevis,omitempty"`
	Device     *string      `json:"device,omitempty"`
	KeyFile    Resource     `json:"keyFile,omitempty"`
	Label      *string      `json:"label,omitempty"`
	Name       string       `json:"name"`
	Options    []LuksOption `json:"options,omitempty"`
	UUID       *string      `json:"uuid,omitempty"`
	WipeVolume *bool        `json:"wipeVolume,omitempty"`
}

type LuksOption string

type MountOption string

type NoProxyItem string

type Node struct {
	Group     NodeGroup `json:"group,omitempty"`
	Overwrite *bool     `json:"overwrite,omitempty"`
	Path      string    `json:"path"`
	User      NodeUser  `json:"user,omitempty"`
}

type NodeGroup struct {
	ID   *int    `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type NodeUser struct {
	ID   *int    `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type Partition struct {
	GUID               *string `json:"guid,omitempty"`
	Label              *string `json:"label,omitempty"`
	Number             int     `json:"number,omitempty"`
	Resize             *bool   `json:"resize,omitempty"`
	ShouldExist        *bool   `json:"shouldExist,omitempty"`
	SizeMiB            *int    `json:"sizeMiB,omitempty"`
	StartMiB           *int    `json:"startMiB,omitempty"`
	TypeGUID           *string `json:"typeGuid,omitempty"`
	WipePartitionEntry *bool   `json:"wipePartitionEntry,omitempty"`
}

type Passwd struct {
	Groups []PasswdGroup `json:"groups,omitempty"`
	Users  []PasswdUser  `json:"users,omitempty"`
}

type PasswdGroup struct {
	Gid          *int    `json:"gid,omitempty"`
	Name         string  `json:"name"`
	PasswordHash *string `json:"passwordHash,omitempty"`
	ShouldExist  *bool   `json:"shouldExist,omitempty"`
	System       *bool   `json:"system,omitempty"`
}

type PasswdUser struct {
	Gecos             *string            `json:"gecos,omitempty"`
	Groups            []Group            `json:"groups,omitempty"`
	HomeDir           *string            `json:"homeDir,omitempty"`
	Name              string             `json:"name"`
	NoCreateHome      *bool              `json:"noCreateHome,omitempty"`
	NoLogInit         *bool              `json:"noLogInit,omitempty"`
	NoUserGroup       *bool              `json:"noUserGroup,omitempty"`
	PasswordHash      *string            `json:"passwordHash,omitempty"`
	PrimaryGroup      *string            `json:"primaryGroup,omitempty"`
	SSHAuthorizedKeys []SSHAuthorizedKey `json:"sshAuthorizedKeys,omitempty"`
	Shell             *string            `json:"shell,omitempty"`
	ShouldExist       *bool              `json:"shouldExist,omitempty"`
	System            *bool              `json:"system,omitempty"`
	UID               *int               `json:"uid,omitempty"`
}

type Proxy struct {
	HTTPProxy  *string       `json:"httpProxy,omitempty"`
	HTTPSProxy *string       `json:"httpsProxy,omitempty"`
	NoProxy    []NoProxyItem `json:"noProxy,omitempty"`
}

type Raid struct {
	Devices []Device     `json:"devices"`
	Level   string       `json:"level"`
	Name    string       `json:"name"`
	Options []RaidOption `json:"options,omitempty"`
	Spares  *int         `json:"spares,omitempty"`
}

type RaidOption string

type Resource struct {
	Compression  *string      `json:"compression,omitempty"`
	HTTPHeaders  HTTPHeaders  `json:"httpHeaders,omitempty"`
	Source       *string      `json:"source,omitempty"`
	Verification Verification `json:"verification,omitempty"`
}

type SSHAuthorizedKey string

type Security struct {
	TLS TLS `json:"tls,omitempty"`
}

type Storage struct {
	Directories []Directory  `json:"directories,omitempty"`
	Disks       []Disk       `json:"disks,omitempty"`
	Files       []File       `json:"files,omitempty"`
	Filesystems []Filesystem `json:"filesystems,omitempty"`
	Links       []Link       `json:"links,omitempty"`
	Luks        []Luks       `json:"luks,omitempty"`
	Raid        []Raid       `json:"raid,omitempty"`
}

type Systemd struct {
	Units []Unit `json:"units,omitempty"`
}

type TLS struct {
	CertificateAuthorities []Resource `json:"certificateAuthorities,omitempty"`
}

type Tang struct {
	Thumbprint *string `json:"thumbprint,omitempty"`
	URL        string  `json:"url,omitempty"`
}

type Timeouts struct {
	HTTPResponseHeaders *int `json:"httpResponseHeaders,omitempty"`
	HTTPTotal           *int `json:"httpTotal,omitempty"`
}

type Unit struct {
	Contents *string  `json:"contents,omitempty"`
	Dropins  []Dropin `json:"dropins,omitempty"`
	Enabled  *bool    `json:"enabled,omitempty"`
	Mask     *bool    `json:"mask,omitempty"`
	Name     string   `json:"name"`
}

type Verification struct {
	Hash *string `json:"hash,omitempty"`
}
