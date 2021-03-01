package config

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mevansam/goutils/crypto"
	"github.com/mevansam/goutils/logger"
	"github.com/spf13/viper"

	"github.com/appbricks/cloud-builder/cookbook"
)

type GetPassphrase func() string

type GetSystemPassphrase func() string

// Function to retrieve a passphrase to encrypt
// temporarily saved keys. By default this will
// be the timestamp of the executable for this
// program.
var SystemPassphrase GetSystemPassphrase

type configFile struct {
	viper.Viper

	keyEncryptPassphrase string

	path      string
	timestamp int64

	keyTimeout int64
	passphrase string

	authContext AuthContext
	context Context
}

// initializes file based configuration
//
// in: path - the path of the config file
// in: passphrase - callback to get the passphrase that will be
//                  used for encrytion of sensitive information
// in: cookbook - the embedded cookbook the config should be
//                assciated with
// out: a Config instance containing the global
//      configuration for CloudBuilder
func InitFileConfig(
	path string,
	cookbook *cookbook.Cookbook,
	getPassphrase GetPassphrase,
) (Config, error) {

	var (
		err error

		absPath  string
		fileInfo os.FileInfo
		v        interface{}

		crypt *crypto.Crypt
	)

	config := &configFile{
		Viper: *viper.New(),

		path: path,
	}

	// initialize auth context
	config.authContext = NewAuthContext()

	// initialize cookbook configuration context
	if config.context, err = NewConfigContext(cookbook); err != nil {
		return nil, err
	}

	// initialize and load viper config file
	if absPath, err = filepath.Abs(path); err != nil {
		return nil, err
	}
	configDir := filepath.Dir(absPath)
	configFileName := filepath.Base(absPath)
	configFileExt := filepath.Ext(absPath)
	configName := configFileName[:len(configFileName)-len(configFileExt)]

	config.SetConfigType(configFileExt[1:])
	config.SetConfigName(configName)
	config.AddConfigPath(configDir)

	config.SetDefault("initialized", false)
	config.SetDefault("keyTimeout", -1)

	if err = config.ReadInConfig(); err != nil {

		if err = os.MkdirAll(configDir, os.ModePerm); err != nil {
			return nil, err
		}
		if err = config.WriteConfigAs(absPath); err != nil {
			return nil, err
		}
		logger.TraceMessage(
			"Creating empty config file: %s",
			absPath)
	}
	if err = config.ReadInConfig(); err != nil {
		return nil, err
	}
	config.AutomaticEnv()

	config.keyEncryptPassphrase = SystemPassphrase()
	logger.TraceMessage(
		"Passphrase used to encrypt saved keys is '%s'.",
		config.keyEncryptPassphrase)

	// the last modification time of the configuration
	// file is used as the seed for encryption keys
	if fileInfo, err = os.Stat(config.path); err != nil {
		return nil, err
	}
	configFileModeTime := fileInfo.ModTime()
	logger.TraceMessage(
		"Config path '%s' with timestamp of '%s'.",
		config.path, configFileModeTime.String())

	config.timestamp = configFileModeTime.UnixNano()

	// retrieve key expiration
	config.keyTimeout = config.GetInt64("keyTimeout")

	// retrieve saved passphrase from config file if it has not expired
	v = config.Get("key")
	if v != nil && time.Now().Local().UnixNano() < (config.timestamp+config.keyTimeout) {

		if crypt, err = crypto.NewCrypt(
			crypto.KeyFromPassphrase(
				config.keyEncryptPassphrase,
				config.timestamp,
			),
		); err != nil {
			return nil, err
		}
		if config.passphrase, err = crypt.DecryptB64(v.(string)); err != nil {

			logger.TraceMessage(
				"Unable to decrypt saved passphrase. Most likely the key has expired: %s",
				err.Error())

			config.passphrase = getPassphrase()
		}

	} else if config.keyTimeout != -1 {
		config.passphrase = getPassphrase()
	}

	logger.DebugMessage("Using config file: %s", config.ConfigFileUsed())
	return config, nil
}

func (cf *configFile) Reset() error {

	var (
		err error
	)

	cf.authContext.Reset()
	if err = cf.context.Reset(); err != nil {
		return err
	}

	cf.Set("initialized", false)
	cf.Set("keyTimeout", -1)

	cf.keyTimeout = -1
	cf.passphrase = ""

	return nil
}

func (cf *configFile) Load() error {

	var (
		err error

		crypt         *crypto.Crypt
		contextReader io.Reader
	)

	// if passphrase is set then create an 
	// crypto function from it
	if len(cf.passphrase) > 0 {
		if crypt, err = crypto.NewCrypt(
			crypto.KeyFromPassphrase(cf.passphrase, cf.timestamp),
		); err != nil {
			return err
		}	
	}

	// get value from config file
	var getValue = func(key string) (io.Reader, error) {
		value := cf.Get(key)
		if value != nil {
			if crypt != nil {
				var decryptedContext string

				if decryptedContext, err = crypt.DecryptB64(value.(string)); err != nil {
					return nil, err
				}
				logger.TraceMessage("Loading serialized value for key %s: %s", key, decryptedContext)
				return strings.NewReader(decryptedContext), nil
	
			} else {
				var encodedContext []byte

				if encodedContext, err = base64.URLEncoding.DecodeString(value.(string)); err != nil {
					return nil, err
				}
				return bytes.NewReader(encodedContext), nil
			}
		}
		return nil, nil
	}

	// load auth context
	if contextReader, err = getValue("authContext"); err != nil {
		return err
	}
	if contextReader != nil {
		if err = cf.authContext.Load(contextReader); err != nil {
			return err
		}
	}

	// load config context
	if contextReader, err = getValue("context"); err != nil {
		return err
	}
	if contextReader != nil {
		if err = cf.context.Load(contextReader); err != nil {
			return err
		}
	}

	logger.TraceMessage("Config loaded from: %s", cf.path)
	return nil
}

func (cf *configFile) Save() error {

	var (
		err error

		crypt *crypto.Crypt
		key   string

		contextOutput strings.Builder		
	)

	// file mod times are in seconds so retrieve
	// timestamp as seconds and convert to nanos
	// for use as the seed
	now := time.Unix(time.Now().Local().Unix(), 0)
	timestamp := now.UnixNano()

	// if passphrase is set then create an 
	// crypto function from it
	if len(cf.passphrase) > 0 {
		if crypt, err = crypto.NewCrypt(
			crypto.KeyFromPassphrase(cf.passphrase, timestamp),
		); err != nil {
			return err
		}	
	}

	// set value in config file 
	var setValue = func(key string, value string) error {
		logger.TraceMessage("Saving serialized value of key \"%s\": %s", key, value)
	
		if crypt != nil {
			var encryptedContext string

			// encrypt auth context
			if encryptedContext, err = crypt.EncryptB64(value); err != nil {
				return err
			}
			cf.Set(key, encryptedContext)
		} else {
			cf.Set(key, base64.URLEncoding.EncodeToString([]byte(value)))
		}
		return nil
	}

	// save auth context
	if err = cf.authContext.Save(&contextOutput); err != nil {
		return err
	}
	if err = setValue("authContext", contextOutput.String()); err != nil {
		return err
	}

	// reset so output buffer can be reused
	contextOutput.Reset()

	// save config context
	if err = cf.context.Save(&contextOutput); err != nil {
		return err
	}
	if err = setValue("context", contextOutput.String()); err != nil {
		return err
	}

	// if the key timeout is set then save the encrypted
	// passphrase. this key will be used to retrieve the
	// context encryption passphrase if config file is
	// read before timeout has expired.
	if cf.keyTimeout > 0 {
		if crypt, err = crypto.NewCrypt(
			crypto.KeyFromPassphrase(
				cf.keyEncryptPassphrase,
				timestamp,
			),
		); err != nil {
			return err
		}
		if key, err = crypt.EncryptB64(cf.passphrase); err != nil {
			return err
		}
		cf.Set("key", key)

	} else {
		cf.Set("key", nil)
	}
	cf.Set("keyTimeout", cf.keyTimeout)

	// save config file
	if err = cf.WriteConfig(); err != nil {
		return err
	}

	// set config file modification time to timestamp
	if err = os.Chtimes(cf.path, now, now); err != nil {
		return err
	}

	logger.TraceMessage("Config saved to: %s (seed time %s)", cf.path, now.String())
	return nil
}

func (cf *configFile) EULAAccepted() bool {
	return cf.GetBool("eulaaccepted")
}

func (cf *configFile) SetEULAAccepted() {
	cf.Set("eulaaccepted", true)
}

func (cf *configFile) Initialized() bool {
	return cf.GetBool("initialized")
}

func (cf *configFile) SetInitialized() {
	cf.Set("initialized", true)
}

func (cf *configFile) HasPassphrase() bool {
	return cf.keyTimeout != -1
}

func (cf *configFile) SetPassphrase(passphrase string) {
	cf.passphrase = passphrase

	if len(passphrase) == 0 {
		cf.keyTimeout = -1
	} else if cf.keyTimeout == -1 {
		cf.keyTimeout = 0
	}
}

func (cf *configFile) SetKeyTimeout(timeout time.Duration) {
	cf.keyTimeout = int64(timeout)
}

func (cf *configFile) AuthContext() AuthContext {
	return cf.authContext
}

func (cf *configFile) Context() Context {
	return cf.context
}

func init() {

	// retrieve the program executable's timestamp
	// to use as the system passphrase to encrypt
	// temporarily saved keys.
	SystemPassphrase = func() string {

		var (
			err error

			absPath  string
			fileInfo os.FileInfo
		)

		// the last modification time of the executable
		if absPath, err = os.Executable(); err != nil {
			panic(err)
		}
		logger.TraceMessage(
			"Will be using modification time from executable '%s' as encryption passphrase.",
			absPath,
		)
		if fileInfo, err = os.Stat(absPath); err != nil {
			panic(err)
		}
		return fileInfo.ModTime().String()
	}
}
