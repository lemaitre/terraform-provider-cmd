package provider

import (
  "errors"
  "strconv"
  "bytes"
  "fmt"
  "encoding/json"

	"golang.org/x/crypto/ssh"
)

type shell_ssh struct {
  client *ssh.Client
}

func (sh *shell_ssh) Execute(command string, env map[string]string) (string, string, error) {
  var stdout, stderr bytes.Buffer
  session, err := sh.client.NewSession()
  if err != nil {
    return "", "", err
  }
  defer session.Close()
  session.Stdout = &stdout
  session.Stderr = &stderr

  for k, v := range env {
    session.Setenv(k, v)
    if err != nil {
      return "", "", err
    }
  }

  err = session.Run(command)
  if err != nil {
    return "", "", err
  }

  return stdout.String(), stderr.String(), nil
}

func (sh *shell_ssh) Close() {
  if sh.client == nil {
    sh.client.Close()
    sh.client = nil
  }
}

func shell_ssh_finalizer(sh *shell_ssh) {
  sh.Close()
}

type ssh_client_cache_type struct {
  clients map[string]*ssh.Client
}
var ssh_client_cache ssh_client_cache_type

func shell_ssh_factory(options map[string]string) (shell, error) {
  if ssh_client_cache.clients == nil {
    ssh_client_cache.clients = make(map[string]*ssh.Client)
  }
  js, _ := json.Marshal(options)
  cache_key := string(js)
  if client, ok := ssh_client_cache.clients[cache_key]; ok {
    return &shell_ssh{
      client: client,
    }, nil
  }


  var err error
  var sh shell_ssh

  var hostname string
  protocol := "tcp"
  port := 22

  config := ssh.ClientConfig{
    User: "root",
    Auth: []ssh.AuthMethod {},
    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
  }

  if hostname_, ok := options["hostname"]; ok {
    hostname = hostname_
  } else {
    return nil, errors.New("Missing hostname")
  }

  if port_, ok := options["port"]; ok {
    port, err = strconv.Atoi(port_)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("\"%s\" is not a valid port number", port_))
    }
  }

  if protocol_, ok := options["protocol"]; ok {
    protocol = protocol_
  }
  if user, ok := options["username"]; ok {
    config.User = user
  }
  if password, ok := options["password"]; ok {
    config.Auth = append(config.Auth, ssh.Password(password))
  }

  sh.client, err = ssh.Dial(protocol, fmt.Sprintf("%s:%d", hostname, port), &config)
  if err != nil {
    return nil, err
  }

  ssh_client_cache.clients[cache_key] = sh.client

  return &sh, nil
}
