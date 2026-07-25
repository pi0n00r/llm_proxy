package handlers

import (
	"net"
	"net/url"
	"strconv"

	"llm_proxy/config"
)

func frontendURL(cfg *config.Config, path string) string {
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Path:   path,
	}).String()
}
