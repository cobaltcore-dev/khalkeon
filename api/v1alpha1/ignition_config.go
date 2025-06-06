// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// this file is copy of "github.com/coreos/ignition/v2/config/v3_5/types/schema.go" with some changes

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Cex struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type Clevis struct {
	Custom    ClevisCustom `json:"custom,omitempty"`
	Tang      []Tang       `json:"tang,omitempty"`
	Threshold *int         `json:"threshold,omitempty"`
	Tpm2      *bool        `json:"tpm2,omitempty"`
}

type ClevisCustom struct {
	Config       *string `json:"config,omitempty"`
	NeedsNetwork *bool   `json:"needsNetwork,omitempty"`
	Pin          *string `json:"pin,omitempty"`
}

type Config struct {
	Ignition        Ignition        `json:"ignition"`
	KernelArguments KernelArguments `json:"kernelArguments,omitempty"`
	Passwd          Passwd          `json:"passwd,omitempty"`
	Storage         Storage         `json:"storage,omitempty"`
	Systemd         Systemd         `json:"systemd,omitempty"`
}

type Device string

type Directory struct {
	Node               `json:"node,omitempty"`
	DirectoryEmbedded1 `json:"directoryEmbedded1,omitempty"`
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
	Node          `json:"node,omitempty"`
	FileEmbedded1 `json:"fileEmbedded1,omitempty"`
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
	Version  string         `json:"version"`
}

type IgnitionConfig struct {
	Merge   *metav1.LabelSelector    `json:"merge,omitempty"`
	Replace *v1.LocalObjectReference `json:"replace,omitempty"`
}

type KernelArgument string

type KernelArguments struct {
	ShouldExist    []KernelArgument `json:"shouldExist,omitempty"`
	ShouldNotExist []KernelArgument `json:"shouldNotExist,omitempty"`
}

type Link struct {
	Node          `json:"node,omitempty"`
	LinkEmbedded1 `json:"linkEmbedded1,omitempty"`
}

type LinkEmbedded1 struct {
	Hard   *bool   `json:"hard,omitempty"`
	Target *string `json:"target,omitempty"`
}

type Luks struct {
	Cex         Cex          `json:"cex,omitempty"`
	Clevis      Clevis       `json:"clevis,omitempty"`
	Device      *string      `json:"device,omitempty"`
	Discard     *bool        `json:"discard,omitempty"`
	KeyFile     Resource     `json:"keyFile,omitempty"`
	Label       *string      `json:"label,omitempty"`
	Name        string       `json:"name"`
	OpenOptions []OpenOption `json:"openOptions,omitempty"`
	Options     []LuksOption `json:"options,omitempty"`
	UUID        *string      `json:"uuid,omitempty"`
	WipeVolume  *bool        `json:"wipeVolume,omitempty"`
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

type OpenOption string

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
	Devices []Device     `json:"devices,omitempty"`
	Level   *string      `json:"level,omitempty"`
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
	Advertisement *string `json:"advertisement,omitempty"`
	Thumbprint    *string `json:"thumbprint,omitempty"`
	URL           string  `json:"url,omitempty"`
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
