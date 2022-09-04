package main

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	http "github.com/kawacode/fhttp"
	http2 "github.com/kawacode/fhttp/http2"
	goproxy "github.com/kawacode/goproxy"
	gostruct "github.com/kawacode/gostruct"
	gotools "github.com/kawacode/gotools"
	tls "github.com/kawacode/utls"
)

// Create a client based on the protocol version
func CreateClient(bot *gostruct.BotData) (*http.Client, error) {
	var client *http.Client
	if bot.HttpRequest.Request.Protocol == "1" {
		var err error
		client, err = CreateHttp1Client(bot)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		client, err = CreateHttp2Client(bot)
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}

// It creates an HTTP/1.1 client with the ability to use a proxy, disable redirects, and set a timeout, that uses a custom TLS dialer that uses a custom JA3 fingerprint
func CreateHttp1Client(bot *gostruct.BotData) (*http.Client, error) {
	http1transport := http.Transport{
		DisableCompression: bot.HttpRequest.Request.DisableCompression,
		DisableKeepAlives:  true,
		ForceAttemptHTTP2:  bot.HttpRequest.Request.HTTP1TRANSPORT.ForceAttemptHTTP2,
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			tls.EnableWeakCiphers()
			var conn net.Conn
			if len(bot.HttpRequest.Request.Proxy) > 0 {
				dialer, err := goproxy.CreateProxyDialer(bot.HttpRequest.Request.Proxy)
				if err != nil {
					return nil, err
				}
				con, err := dialer.Dial(network, addr)
				if err != nil {
					return nil, err
				}
				conn = con
			} else {
				var err error
				conn, err = net.Dial(network, addr)
				if err != nil {
					return nil, err
				}
			}
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			config := &tls.Config{ServerName: host, InsecureSkipVerify: bot.HttpRequest.Request.InsecureSkipVerify}
			var uconn *tls.UConn
			if bot.HttpRequest.Request.HelloClient.Str() != "-" {
				uconn = tls.UClient(conn, config, bot.HttpRequest.Request.HelloClient)
				if strings.Contains(bot.HttpRequest.Request.HelloClient.Str(), "CustomInternal") {
					if bot.HttpRequest.Request.Ja3 == "" {
						return nil, errors.New("missing x-kc-clientspec")
					}
					if bot.HttpRequest.Request.Protocol != "2" && bot.HttpRequest.Request.Protocol != "1" {
						return nil, errors.New("missing x-kc-protocol")
					}
					tlsspec, err := gotools.ParseJA3(bot.HttpRequest.Request.Ja3, bot.HttpRequest.Request.Protocol)
					if err != nil {
						return nil, err
					}
					if err := uconn.ApplyPreset(tlsspec); err != nil {
						return nil, err
					}
					if err := uconn.Handshake(); err != nil {
						return nil, err
					}
				}
			} else {
				uconn = tls.UClient(conn, config, tls.HelloChrome_Auto)
			}
			return uconn, nil
		},
	}
	timeout := gotools.IsInt(bot.HttpRequest.Request.Timeout)
	var client *http.Client
	if timeout {
		timeoutsec, _ := strconv.ParseInt(bot.HttpRequest.Request.Timeout, 0, 64)
		client = &http.Client{
			Transport: &http1transport,
			Timeout:   time.Duration(time.Duration(timeoutsec) * time.Second),
		}
	} else {
		client = &http.Client{
			Transport: &http1transport,
			Timeout:   time.Duration(time.Duration(30) * time.Second),
		}
	}
	if gotools.IsInt(bot.HttpRequest.Request.MaxRedirects) {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			maxredirects, _ := strconv.ParseInt(bot.HttpRequest.Request.MaxRedirects, 0, 16)
			if len(via) >= int(maxredirects) {
				return http.ErrUseLastResponse
			}
			return nil
		}
	} else if bot.HttpRequest.Request.MaxRedirects == "false" {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			maxredirects := 10
			if len(via) >= maxredirects {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}
	return client, nil
}

// It creates an HTTP2 client with the ability to use a proxy, disable redirects, and set a timeout, that uses a custom TLS dialer that uses a custom JA3 fingerprint
func CreateHttp2Client(bot *gostruct.BotData) (*http.Client, error) {
	http2transport := http2.Transport{
		AllowHTTP:                  bot.HttpRequest.Request.HTTP2TRANSPORT.AllowHTTP,
		StrictMaxConcurrentStreams: false,
		DisableCompression:         bot.HttpRequest.Request.DisableCompression,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			tls.EnableWeakCiphers()
			var conn net.Conn
			if len(bot.HttpRequest.Request.Proxy) > 0 {
				dialer, err := goproxy.CreateProxyDialer(bot.HttpRequest.Request.Proxy)
				if err != nil {
					return nil, err
				}
				con, err := dialer.Dial(network, addr)
				if err != nil {
					return nil, err
				}
				conn = con
			} else {
				var err error
				conn, err = net.Dial(network, addr)
				if err != nil {
					return nil, err
				}
			}
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			config := &tls.Config{ServerName: host, InsecureSkipVerify: bot.HttpRequest.Request.InsecureSkipVerify}
			var uconn *tls.UConn
			if bot.HttpRequest.Request.HelloClient.Str() != "-" {
				uconn = tls.UClient(conn, config, bot.HttpRequest.Request.HelloClient)
				if strings.Contains(bot.HttpRequest.Request.HelloClient.Str(), "CustomInternal") {
					if bot.HttpRequest.Request.Ja3 == "" {
						return nil, errors.New("missing x-kc-clientspec")
					}
					if bot.HttpRequest.Request.Protocol != "2" && bot.HttpRequest.Request.Protocol != "1" {
						return nil, errors.New("missing x-kc-protocol")
					}
					tlsspec, err := gotools.ParseJA3(bot.HttpRequest.Request.Ja3, bot.HttpRequest.Request.Protocol)
					if err != nil {
						return nil, err
					}
					if err := uconn.ApplyPreset(tlsspec); err != nil {
						return nil, err
					}
					if err := uconn.Handshake(); err != nil {
						return nil, err
					}
				}
			} else {
				uconn = tls.UClient(conn, config, tls.HelloChrome_Auto)
			}
			return uconn, nil
		},
	}
	Settings, SettingsOrder, Priorities, windowupdate := gotools.GetFrameSettingsStringList(bot)
	http2transport.SettingsOrder = SettingsOrder
	http2transport.Settings = Settings
	http2transport.Priorities = Priorities
	http2transport.ConnectionFlow = uint32(windowupdate)
	http2transport.PushHandler = &http2.DefaultPushHandler{}
	timeout := gotools.IsInt(bot.HttpRequest.Request.Timeout)
	var client *http.Client
	if timeout {
		timeoutsec, _ := strconv.ParseInt(bot.HttpRequest.Request.Timeout, 0, 64)
		client = &http.Client{
			Transport: &http2transport,
			Timeout:   time.Duration(time.Duration(timeoutsec) * time.Second),
		}
	} else {
		client = &http.Client{
			Transport: &http2transport,
			Timeout:   time.Duration(time.Duration(30) * time.Second),
		}
	}
	if gotools.IsInt(bot.HttpRequest.Request.MaxRedirects) {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			maxredirects, _ := strconv.ParseInt(bot.HttpRequest.Request.MaxRedirects, 0, 16)
			if len(via) >= int(maxredirects) {
				return http.ErrUseLastResponse
			}
			return nil
		}
	} else if bot.HttpRequest.Request.MaxRedirects == "false" {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			maxredirects := 10
			if len(via) >= maxredirects {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}
	return client, nil
}
