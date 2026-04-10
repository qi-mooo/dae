package api

import (
	"errors"
	"time"
)

var ErrProviderNotFound = errors.New("not found")

type Traffic struct {
	Up        int64 `json:"up"`
	Down      int64 `json:"down"`
	UpTotal   int64 `json:"upTotal"`
	DownTotal int64 `json:"downTotal"`
}

type ConnectionsSnapshot struct {
	UpdatedAt   time.Time    `json:"updatedAt"`
	Total       int          `json:"total"`
	TCP         int          `json:"tcp"`
	UDP         int          `json:"udp"`
	Connections []Connection `json:"connections"`
}

type Connection struct {
	ID                 string    `json:"id"`
	Network            string    `json:"network"`
	State              string    `json:"state"`
	Source             string    `json:"source"`
	SourceAddress      string    `json:"sourceAddress"`
	SourcePort         uint16    `json:"sourcePort"`
	Destination        string    `json:"destination"`
	DestinationAddress string    `json:"destinationAddress"`
	DestinationPort    uint16    `json:"destinationPort"`
	Process            string    `json:"process"`
	PID                uint32    `json:"pid"`
	Outbound           string    `json:"outbound"`
	Direction          string    `json:"direction"`
	Mark               uint32    `json:"mark"`
	DSCP               uint8     `json:"dscp"`
	Must               bool      `json:"must"`
	HasRouting         bool      `json:"hasRouting"`
	Mac                string    `json:"mac"`
	LastSeen           time.Time `json:"lastSeen"`
	UploadSpeed        int64     `json:"uploadSpeed"`
	DownloadSpeed      int64     `json:"downloadSpeed"`
	UploadTotal        int64     `json:"uploadTotal"`
	DownloadTotal      int64     `json:"downloadTotal"`
}

type Memory struct {
	Inuse   uint64 `json:"inuse"`
	RSS     uint64 `json:"rss"`
	OSLimit uint64 `json:"oslimit"`
}

type Config struct {
	Port        int    `json:"port"`
	SocksPort   int    `json:"socks-port"`
	RedirPort   int    `json:"redir-port"`
	TProxyPort  int    `json:"tproxy-port"`
	MixedPort   int    `json:"mixed-port"`
	AllowLan    bool   `json:"allow-lan"`
	BindAddress string `json:"bind-address"`
	Mode        string `json:"mode"`
	LogLevel    string `json:"log-level"`
	IPv6        bool   `json:"ipv6"`
}

type DaeConfigDocument struct {
	Path      string          `json:"path"`
	Content   string          `json:"content"`
	Documents []DaeConfigFile `json:"documents,omitempty"`
}

type DaeConfigFile struct {
	Path         string `json:"path"`
	RelativePath string `json:"relativePath,omitempty"`
	Content      string `json:"content"`
	Entry        bool   `json:"entry,omitempty"`
	Missing      bool   `json:"missing,omitempty"`
}

type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay uint16    `json:"delay"`
}

type Proxy struct {
	Name            string         `json:"name"`
	Type            string         `json:"type"`
	Alive           bool           `json:"alive"`
	History         []DelayHistory `json:"history"`
	Now             string         `json:"now,omitempty"`
	All             []string       `json:"all,omitempty"`
	UDP             bool           `json:"udp"`
	XUDP            bool           `json:"xudp"`
	Address         string         `json:"addr,omitempty"`
	Protocol        string         `json:"protocol,omitempty"`
	SubscriptionTag string         `json:"subscriptionTag,omitempty"`
}

type Provider interface {
	Hello() string
	Version() string
	Meta() bool
	Config() Config
	DaeConfigDocument() (DaeConfigDocument, error)
	Memory() Memory
	Traffic() Traffic
	Connections(limit int) ConnectionsSnapshot
	Proxies() map[string]Proxy
	Proxy(name string) (Proxy, bool)
	UpdateProxy(groupName, proxyName string) error
	ResetProxy(groupName string) error
	Delay(name, probeURL string, timeout time.Duration) (int, error)
	SetLogLevel(level string) error
	UpdateDaeConfig(document DaeConfigDocument) error
}
