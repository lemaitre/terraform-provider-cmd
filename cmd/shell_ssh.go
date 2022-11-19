package cmd

import (
  "context"
  "fmt"
  "encoding/pem"
  "crypto/x509"
  "io/ioutil"

	"golang.org/x/crypto/ssh"

  "github.com/hashicorp/terraform-plugin-framework/diag"
  "github.com/hashicorp/terraform-plugin-framework/tfsdk"
  "github.com/hashicorp/terraform-plugin-framework/types"
)

type shellSsh struct {
  client *ssh.Client
}

var shellSshFactory shellFactory = shellFactory{
  IsRemote: true,
  Name: "ssh",
  Schema: map[string]tfsdk.Attribute{
    "hostname": tfsdk.Attribute{
      Type: types.StringType,
      Description: "Hostname used for the ssh connection",
      Required: true,
    },
    "port": tfsdk.Attribute{
      Type: types.NumberType,
      Description: "Port used for the ssh connection (default: 22)",
      Optional: true,
    },
    "protocol": tfsdk.Attribute{
      Type: types.StringType,
      Description: "Protocol used for the ssh connection (default: \"tcp\")",
      Optional: true,
    },
    "username": tfsdk.Attribute{
      Type: types.StringType,
      Description: "Username used for the ssh connection (default: \"root\")",
      Optional: true,
    },
    "password": tfsdk.Attribute{
      Type: types.StringType,
      Description: "Password used for the ssh connection",
      Optional: true,
      Sensitive: true,
    },
    "key": tfsdk.Attribute{
      Type: types.StringType,
      Description: "ssh key",
      Optional: true,
      Sensitive: true,
    },
    "keyfile": tfsdk.Attribute{
      Type: types.StringType,
      Description: "Path to the ssh key",
      Optional: true,
    },
    "keypassword": tfsdk.Attribute{
      Type: types.StringType,
      Description: "Password used to decrypt the ssh key",
      Optional: true,
      Sensitive: true,
    },
  },
  Create: func (ctx context.Context, val types.Object) (shell, diag.Diagnostics) {
    if sshCache.clients == nil {
      sshCache.clients = make(map[string]*ssh.Client)
    }
    cacheKey := val.String()
    if client, ok := sshCache.clients[cacheKey]; ok {
      return &shellSsh{
        client: client,
      }, nil
    }

    type connectionModel struct {
      Hostname    string `tfsdk:"hostname"`
      Port        int    `tfsdk:"port"`
      Protocol    string `tfsdk:"protocol"`
      Username    string `tfsdk:"username"`
      Password    string `tfsdk:"password"`
      Key         string `tfsdk:"key"`
      Keyfile     string `tfsdk:"keyfile"`
      Keypassword string `tfsdk:"keypassword"`
    }

    var connection connectionModel
    diags := val.As(ctx, &connection, types.ObjectAsOptions{true, true})

    if len(diags) > 0 {
      return nil, diags
    }

    if connection.Hostname == "" {
      return nil, diag.Diagnostics{
        diag.NewErrorDiagnostic(
          "Missing Hostname",
          "The hostname for the ssh connection is missing",
        ),
      }
    }
    if connection.Port == 0 {
      connection.Port = 22
    }
    if connection.Username == "" {
      connection.Username = "root"
    }
    if connection.Protocol == "" {
      connection.Protocol = "tcp"
    }

    var err error
    var sh shellSsh

    config := ssh.ClientConfig{
      User: connection.Username,
      Auth: []ssh.AuthMethod {},
      HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }
    if connection.Password != "" {
      config.Auth = append(config.Auth, ssh.Password(connection.Password))
    }

    var key []byte
    if connection.Keyfile != "" {
      key, err = ioutil.ReadFile(connection.Keyfile)
      if err != nil {
        return nil, diag.Diagnostics{
          diag.NewErrorDiagnostic(
            "Error while reading keyfile",
            fmt.Sprintf("%s", err),
          ),
        }
      }
    }
    if connection.Key != "" {
      key = []byte(connection.Key)
    }

    if key != nil {
      signer, err := signerFromPem(key, connection.Keypassword)
      if err != nil {
        return nil, diag.Diagnostics{
          diag.NewErrorDiagnostic(
            "Error while parsing key",
            fmt.Sprintf("%s", err),
          ),
        }
      }
      config.Auth = append(config.Auth, ssh.PublicKeys(signer))
    }

    sh.client, err = ssh.Dial(connection.Protocol, fmt.Sprintf("%s:%d", connection.Hostname, connection.Port), &config)
    if err != nil {
      return nil, diag.Diagnostics{
        diag.NewErrorDiagnostic(
          "Error during the ssh connection",
          fmt.Sprintf("%s", err),
        ),
      }
    }

    sshCache.clients[cacheKey] = sh.client

    return &sh, nil
  },
}

func (sh *shellSsh) Execute(command string, env map[string]string) (string, string, string, error) {
  out := NewCommandOutput()
  session, err := sh.client.NewSession()
  if err != nil {
    return "", "", "", err
  }
  defer session.Close()
  session.Stdout = out.StdoutWriter
  session.Stderr = out.StderrWriter

  cmd := "set +v\n"
  for k, v := range env {
    cmd += fmt.Sprintf("IFS= read -r -d '' %s << '__!@#$END_OF_VARIABLE$#@!__' || true\n%s\n__!@#$END_OF_VARIABLE$#@!__\nexport %s=\"${%s%%?}\"\n", k, v, k, k)
    //session.Setenv(k, v)
    //if err != nil {
    //  return "", "", err
    //}
  }
  cmd += command

  err = session.Run(cmd)

  return out.Stdout.String(), out.Stderr.String(), out.Combined.String(), err
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
