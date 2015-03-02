
package main

import (
  "io"
  "fmt"
  "log"
  "strings"
  "net/http"
  "encoding/base64"

  "github.com/konek/auth-lib"
)

type handler struct {
  Conf config
  Auth auth.Auth
}

func splitPath(path string) []string {
  if path[0] == '/' {
    path = path[1:]
  }
  if len(path) == 0 {
    return nil
  }
  if path[len(path)-1] == '/' {
    path = path[0:len(path)-1]
  }
  if len(path) == 0 {
    return nil
  }
  return strings.Split(path, "/")
}

func comparePaths(a []string, b []string) bool {
  if len(b) < len(a) {
    return false
  }
  for i := 0; i < len(a); i++ {
    if a[i] != b[i] {
      return false
    }
  }
  return true
}

func (h handler) pass(w http.ResponseWriter, r *http.Request, loc location) error {
  var authHeaderOk = false
  var uid string

  authHeader := r.Header.Get("Authorization")
  if authHeader != "" {
    header := strings.Split(authHeader, " ")
    if len(header) == 2 && header[0] == "Basic" {
      b, err := base64.StdEncoding.DecodeString(header[1])
      if err != nil {
        return err
      }
      infos := strings.SplitN(string(b), ":", 2)
      if len(infos) == 2 {
        authHeaderOk, uid, err = h.Auth.Auth(loc.Domain, infos[0], infos[1])
      }
    }
  }
  if authHeaderOk == false {
    var realm = loc.Realm
    if realm == "" {
      realm = h.Conf.Realm
    }

    w.Header().Add("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
    w.WriteHeader(401)
    _, err := w.Write([]byte("forbidden access"))
    if err != nil {
      return err
    }
    return nil
  }
  client := &http.Client{}
  url := r.URL
  if loc.StripPath == true {
    url.Path = url.Path[len(loc.Path):]
  }
  req, err := http.NewRequest(r.Method, loc.ProxyPass + url.String(), r.Body)
  if err != nil {
    return err
  }
  req.Header = r.Header
  req.Header.Del("Authorization")
  if loc.PassUID == true {
    req.Header.Add("X-Uid", uid)
  }
  res, err := client.Do(req)
  if err != nil {
    return err
  }
  for k, v := range res.Header {
    if k == "Connection" || len(v) == 0 {
      continue
    }
    w.Header().Set(k, v[0])
    for i := 1; i < len(v); i++ {
      w.Header().Add(k, v[i])
    }
  }
  w.WriteHeader(res.StatusCode)
  _, err = io.Copy(w, res.Body)
  if err != nil {
    return err
  }
  return nil
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  var loc location
  var found = false

  path := splitPath(r.URL.Path)
  for _, l := range h.Conf.Location {
    p := splitPath(l.Path)
    if len(p) == 0 && len(path) == 0 {
      found = true
      loc = l
      break
    }
    if comparePaths(p, path) == true {
      found = true
      loc = l
      break
    }
  }
  if found == false {
    w.WriteHeader(404)
    _, err := w.Write([]byte("page not found"))
    if err != nil {
      log.Println(err)
    }
    return
  }
  err := h.pass(w, r, loc)
  if err != nil {
    log.Println(err)
    w.WriteHeader(500)
    _, err = w.Write([]byte("an unexpected error occured, please contact an administrator"))
    if err != nil {
      log.Println(err)
    }
  }
}

func main() {
  var h handler

  conf, err := readConfig()
  if err != nil {
    panic(err)
  }
  h.Conf = conf
  h.Auth = auth.Auth{
    conf.Auth,
  }
  log.Println("listening on", conf.Listen)
  // TODO ETCD here
  log.Fatal(http.ListenAndServe(conf.Listen, h))
}

