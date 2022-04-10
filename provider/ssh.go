package provider

import (
  "strconv"
  "bytes"
  "fmt"
  "encoding/json"
  "encoding/pem"
  "crypto/x509"
  "io/ioutil"

	"golang.org/x/crypto/ssh"
)

type shellSsh struct {
  client *ssh.Client
}

func (sh *shellSsh) Execute(command string, env map[string]string) (string, string, error) {
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

func (sh *shellSsh) Close() {
  if sh.client == nil {
    sh.client.Close()
    sh.client = nil
  }
}

type sshCacheType struct {
  clients map[string]*ssh.Client
}
var sshCache sshCacheType

func signerFromPem(pemBytes []byte, password string) (ssh.Signer, error) {

  // read pem block
  err := fmt.Errorf("PEM decode failed, no key found")
  pemBlock, _ := pem.Decode(pemBytes)
  if pemBlock == nil {
    return nil, err
  }

  // handle encrypted key
  if x509.IsEncryptedPEMBlock(pemBlock) {
    // decrypt PEM
    pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, []byte(password))
    if err != nil {
      return nil, fmt.Errorf("Decrypting PEM block failed %v", err)
    }

    // get RSA, EC or DSA key
    key, err := parsePemBlock(pemBlock)
    if err != nil {
      return nil, err
    }

    // generate signer instance from key
    signer, err := ssh.NewSignerFromKey(key)
    if err != nil {
      return nil, fmt.Errorf("Creating signer from encrypted key failed %v", err)
    }

    return signer, nil
  } else {
    if password == "" {
      // generate signer instance from plain key
      signer, err := ssh.ParsePrivateKey(pemBytes)
      if err != nil {
        return nil, fmt.Errorf("Parsing plain private key failed %v", err)
      }

      return signer, nil
    } else {
      // generate signer instance from password protected key
      signer, err := ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(password))
      if err != nil {
        return nil, fmt.Errorf("Parsing protected private key failed %v", err)
      }

      return signer, nil
    }
  }
}

func parsePemBlock(block *pem.Block) (interface{}, error) {
  switch block.Type {
  case "RSA PRIVATE KEY":
    key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
    if err != nil {
      return nil, fmt.Errorf("Parsing PKCS private key failed %v", err)
    } else {
      return key, nil
    }
  case "EC PRIVATE KEY":
    key, err := x509.ParseECPrivateKey(block.Bytes)
    if err != nil {
      return nil, fmt.Errorf("Parsing EC private key failed %v", err)
    } else {
      return key, nil
    }
  case "DSA PRIVATE KEY":
    key, err := ssh.ParseDSAPrivateKey(block.Bytes)
    if err != nil {
      return nil, fmt.Errorf("Parsing DSA private key failed %v", err)
    } else {
      return key, nil
    }
  default:
    return nil, fmt.Errorf("Parsing private key failed, unsupported key type %q", block.Type)
  }
}

func shellSshFactory(options map[string]string) (shell, error) {
  if sshCache.clients == nil {
    sshCache.clients = make(map[string]*ssh.Client)
  }
  js, _ := json.Marshal(options)
  cacheKey := string(js)
  if client, ok := sshCache.clients[cacheKey]; ok {
    return &shellSsh{
      client: client,
    }, nil
  }


  var err error
  var sh shellSsh

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
    return nil, fmt.Errorf("Missing hostname")
  }

  if port_, ok := options["port"]; ok {
    port, err = strconv.Atoi(port_)
    if err != nil {
      return nil, fmt.Errorf("\"%s\" is not a valid port number", port_)
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

  var key []byte
  if keyfile, ok := options["keyfile"]; ok {
    key, err = ioutil.ReadFile(keyfile)
    if err != nil {
      return nil, err
    }
  }
  if key_, ok := options["key"]; ok {
    key = []byte(key_)
  }

  if key != nil {
    keypassword := ""
    if password, ok := options["keypassword"]; ok {
      keypassword = password
    }
    signer, err := signerFromPem(key, keypassword)
    if err != nil {
      return nil, err
    }
    config.Auth = append(config.Auth, ssh.PublicKeys(signer))
  }

  sh.client, err = ssh.Dial(protocol, fmt.Sprintf("%s:%d", hostname, port), &config)
  if err != nil {
    return nil, err
  }

  sshCache.clients[cacheKey] = sh.client

  return &sh, nil
}
