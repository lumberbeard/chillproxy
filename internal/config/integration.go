package config

import (
	"net/url"
	"time"
)

// Integration config stub types for existing StremThru integrations
type IntegrationAniListConfig struct {
	ListStaleTime time.Duration
}

type IntegrationBitmagnetConfig struct {
	BaseURL     *url.URL
	DatabaseURI string
}

func (c *IntegrationBitmagnetConfig) IsEnabled() bool {
	return c != nil && c.BaseURL != nil
}

type IntegrationGitHubConfig struct {
	User  string
	Token string
}

func (c *IntegrationGitHubConfig) HasDefaultCredentials() bool {
	return c != nil && c.Token != ""
}

type IntegrationKitsuConfig struct {
	ClientId     string
	ClientSecret string
	Email        string
	Password     string
}

func (c *IntegrationKitsuConfig) HasDefaultCredentials() bool {
	return c != nil && c.Email != "" && c.Password != ""
}

type IntegrationLetterboxdConfig struct {
	ClientId      string
	ClientSecret  string
	UserAgent     string
	ListStaleTime time.Duration
}

func (c *IntegrationLetterboxdConfig) IsEnabled() bool {
	return c != nil && c.ClientId != "" && c.ClientSecret != ""
}

func (c *IntegrationLetterboxdConfig) IsPiggybacked() bool {
	return false
}

type IntegrationMDBListConfig struct {
	ListStaleTime time.Duration
}

type IntegrationTMDBConfig struct {
	AccessToken   string
	ListStaleTime time.Duration
}

func (c *IntegrationTMDBConfig) IsEnabled() bool {
	return c != nil && c.AccessToken != ""
}

type IntegrationTraktConfig struct {
	ClientId      string
	ClientSecret  string
	ListStaleTime time.Duration
}

func (c *IntegrationTraktConfig) IsEnabled() bool {
	return c != nil && c.ClientId != ""
}

type IntegrationTVDBConfig struct {
	APIKey               string
	SystemOAuthTokenId   string
	ListStaleTime        time.Duration
}

func (c *IntegrationTVDBConfig) IsEnabled() bool {
	return c != nil && c.APIKey != ""
}

// Main integration configuration struct
type IntegrationConfig struct {
	AniList    IntegrationAniListConfig
	Bitmagnet  IntegrationBitmagnetConfig
	GitHub     IntegrationGitHubConfig
	Kitsu      IntegrationKitsuConfig
	Letterboxd IntegrationLetterboxdConfig
	MDBList    IntegrationMDBListConfig
	TMDB       IntegrationTMDBConfig
	Trakt      IntegrationTraktConfig
	TVDB       IntegrationTVDBConfig
}

// Global Integration variable (initialized in config.go)
var Integration *IntegrationConfig

// Chillstreams integration config variables
var (
	ChillstreamsAPIURL       string
	ChillstreamsAPIKey       string
	EnableChillstreamsAuth   bool
)

