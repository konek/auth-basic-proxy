
package main

import (
  "os"
  "errors"

  "github.com/BurntSushi/toml"
)

const (
  dListen = ":8000"
  dAuth = "http://localhost:8080"
  dEtcdURL = ""
  dEtcdNode = "auth/basic-proxy"
)

type etcdConfig struct {
  URL   string `toml:"url"`
  Node  string `toml:"node"`
}

type location struct {
  Path      string `toml:"path"`
  Domain    string `toml:"domain"`
  ProxyPass string `toml:"proxy_pass"`
  Realm     string `toml:"realm"`
  PassUID   bool   `toml:"pass_uid"`
  StripPath bool   `toml:"strip_path"`
}

type config struct {
  Listen      string `toml:"listen"`
  Auth       string `toml:"auth"`
  Realm       string `toml:"realm"`
  Etcd        etcdConfig `toml:"etcd"` // TODO
  Location    []location `toml:"location"`
}

func checkConfig(conf config) error {
  if conf.Listen == "" {
    return errors.New("listen can't be empty")
  }
  if conf.Auth == "" {
    return errors.New("Auth can't be empty")
  }
  if conf.Etcd.URL != "" && conf.Etcd.Node == "" {
    return errors.New("etcd.node can't be empty if etcd.url is specified")
  }
  if len(conf.Location) == 0 {
    return errors.New("you must specify at least one location")
  }
  for i, loc := range conf.Location {
    if loc.Path == "" {
      return errors.New("location.path can't be empty")
    }
    if loc.Domain == "" {
      return errors.New("location.domain can't be empty")
    }
    if loc.ProxyPass == "" {
      return errors.New("location.proxy_pass can't be empty")
    }
    if conf.Realm == "" && loc.Realm == "" {
      return errors.New("a realm should be specified either in location or in global configuration")
    }
    for j, loc2 := range conf.Location {
      if i == j {
        continue
      }
      if loc.Path == loc2.Path {
        return errors.New("duplicate location.path")
      }
    }
  }
  return nil
}

func readConfig() (config, error) {
  var ret = config{
    Listen: dListen,
    Auth: dAuth,
    Etcd: etcdConfig{
      URL: dEtcdURL,
      Node: dEtcdNode,
    },
  }

  _, err := toml.DecodeFile("config.toml", &ret)
  if err != nil {
    return ret, err
  }

  if os.Getenv("ETCD") != "" {
    ret.Etcd.URL = os.Getenv("ETCD")
  }
  if os.Getenv("NODE") != "" {
    ret.Etcd.Node = os.Getenv("NODE")
  }
  if os.Getenv("AUTH") != "" {
    ret.Auth = os.Getenv("AUTH")
  }
  if os.Getenv("LISTEN") != "" {
    ret.Listen = os.Getenv("LISTEN")
  }

  err = checkConfig(ret)
  if err != nil {
    return ret, err
  }

  return ret, nil
}

