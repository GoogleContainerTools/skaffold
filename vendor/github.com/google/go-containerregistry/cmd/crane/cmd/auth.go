// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/spf13/cobra"
)

// NewCmdAuth creates a new cobra.Command for the auth subcommand.
func NewCmdAuth(options []crane.Option, argv ...string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Log in or access credentials",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Usage() },
	}
	cmd.AddCommand(NewCmdAuthGet(options, argv...), NewCmdAuthLogin(argv...), NewCmdAuthLogout(argv...), NewCmdAuthToken(options))
	return cmd
}

func NewCmdAuthToken(options []crane.Option) *cobra.Command {
	var (
		header bool
		push   bool
		mounts []string
	)
	cmd := &cobra.Command{
		Use:   "token REPO",
		Short: "Retrieves a token for a remote repo",
		Example: `# If you wanted to mount a blob from debian to ubuntu.
$ curl -H "$(crane auth token -H --push --mount debian ubuntu)" ...

# To get the raw list tags response
$ curl -H "$(crane auth token -H ubuntu)" https://index.docker.io/v2/library/ubuntu/tags/list
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := name.NewRepository(args[0])
			if err != nil {
				return err
			}
			o := crane.GetOptions(options...)

			t := transport.NewLogger(o.Transport)
			pr, err := transport.Ping(cmd.Context(), repo.Registry, t)
			if err != nil {
				return err
			}

			auth, err := authn.Resolve(cmd.Context(), o.Keychain, repo)
			if err != nil {
				return err
			}

			scopes := []string{repo.Scope(transport.PullScope)}
			if push {
				scopes[0] = repo.Scope(transport.PushScope)
			}

			for _, m := range mounts {
				mr, err := name.NewRepository(m)
				if err != nil {
					return err
				}
				scopes = append(scopes, mr.Scope(transport.PullScope))
			}

			tr, err := transport.Exchange(cmd.Context(), repo.Registry, auth, t, scopes, pr)
			if err != nil {
				return err
			}

			if header {
				fmt.Fprintf(cmd.OutOrStdout(), "Authorization: Bearer %s", tr.Token)
				return nil
			}

			if err := json.NewEncoder(os.Stdout).Encode(tr); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&mounts, "mount", "m", []string{}, "Scopes to mount from")
	cmd.Flags().BoolVarP(&header, "header", "H", false, "Output in header format")
	cmd.Flags().BoolVar(&push, "push", false, "Request push scopes")
	return cmd
}

type credentials struct {
	Username string
	Secret   string
}

// https://github.com/docker/cli/blob/2291f610ae73533e6e0749d4ef1e360149b1e46b/cli/config/credentials/native_store.go#L100-L109
func toCreds(config *authn.AuthConfig) credentials {
	creds := credentials{
		Username: config.Username,
		Secret:   config.Password,
	}

	if config.IdentityToken != "" {
		creds.Username = "<token>"
		creds.Secret = config.IdentityToken
	}
	return creds
}

// NewCmdAuthGet creates a new `crane auth get` command.
func NewCmdAuthGet(options []crane.Option, argv ...string) *cobra.Command {
	if len(argv) == 0 {
		argv = []string{os.Args[0]}
	}

	baseCmd := strings.Join(argv, " ")
	eg := fmt.Sprintf(`  # Read configured credentials for reg.example.com
  $ echo "reg.example.com" | %s get
  {"username":"AzureDiamond","password":"hunter2"}
  # or
  $ %s get reg.example.com
  {"username":"AzureDiamond","password":"hunter2"}`, baseCmd, baseCmd)

	return &cobra.Command{
		Use:     "get [REGISTRY_ADDR]",
		Short:   "Implements a credential helper",
		Example: eg,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registryAddr := ""
			if len(args) == 1 {
				registryAddr = args[0]
			} else {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				registryAddr = strings.TrimSpace(string(b))
			}

			reg, err := name.NewRegistry(registryAddr)
			if err != nil {
				return err
			}
			authorizer, err := authn.Resolve(cmd.Context(), crane.GetOptions(options...).Keychain, reg)
			if err != nil {
				return err
			}

			// If we don't find any credentials, there's a magic error to return:
			//
			// https://github.com/docker/docker-credential-helpers/blob/f78081d1f7fef6ad74ad6b79368de6348386e591/credentials/error.go#L4-L6
			// https://github.com/docker/docker-credential-helpers/blob/f78081d1f7fef6ad74ad6b79368de6348386e591/credentials/credentials.go#L61-L63
			if authorizer == authn.Anonymous {
				fmt.Fprint(os.Stdout, "credentials not found in native keychain\n")
				os.Exit(1)
			}

			auth, err := authn.Authorization(cmd.Context(), authorizer)
			if err != nil {
				return err
			}

			// Convert back to a form that credential helpers can parse so that this
			// can act as a meta credential helper.
			creds := toCreds(auth)
			return json.NewEncoder(os.Stdout).Encode(creds)
		},
	}
}

// NewCmdAuthLogin creates a new `crane auth login` command.
func NewCmdAuthLogin(argv ...string) *cobra.Command {
	var opts loginOptions

	if len(argv) == 0 {
		argv = []string{os.Args[0]}
	}

	eg := fmt.Sprintf(`  # Log in to reg.example.com
  %s login reg.example.com -u AzureDiamond -p hunter2`, strings.Join(argv, " "))

	cmd := &cobra.Command{
		Use:     "login [OPTIONS] [SERVER]",
		Short:   "Log in to a registry",
		Example: eg,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			reg, err := name.NewRegistry(args[0])
			if err != nil {
				return err
			}

			opts.serverAddress = reg.Name()

			return login(opts)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.user, "username", "u", "", "Username")
	flags.StringVarP(&opts.password, "password", "p", "", "Password")
	flags.BoolVarP(&opts.passwordStdin, "password-stdin", "", false, "Take the password from stdin")

	return cmd
}

type loginOptions struct {
	serverAddress string
	user          string
	password      string
	passwordStdin bool
}

func login(opts loginOptions) error {
	if opts.passwordStdin {
		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		opts.password = strings.TrimSuffix(string(contents), "\n")
		opts.password = strings.TrimSuffix(opts.password, "\r")
	}
	if opts.user == "" && opts.password == "" {
		return errors.New("username and password required")
	}
	cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
	if err != nil {
		return err
	}
	creds := cf.GetCredentialsStore(opts.serverAddress)
	if opts.serverAddress == name.DefaultRegistry {
		opts.serverAddress = authn.DefaultAuthKey
	}
	if err := creds.Store(types.AuthConfig{
		ServerAddress: opts.serverAddress,
		Username:      opts.user,
		Password:      opts.password,
	}); err != nil {
		return err
	}

	if err := cf.Save(); err != nil {
		return err
	}
	log.Printf("logged in via %s", cf.Filename)
	return nil
}

// NewCmdAuthLogout creates a new `crane auth logout` command.
func NewCmdAuthLogout(argv ...string) *cobra.Command {
	eg := fmt.Sprintf(`  # Log out of reg.example.com
  %s logout reg.example.com`, strings.Join(argv, " "))

	cmd := &cobra.Command{
		Use:     "logout [SERVER]",
		Short:   "Log out of a registry",
		Example: eg,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			reg, err := name.NewRegistry(args[0])
			if err != nil {
				return err
			}
			serverAddress := reg.Name()

			cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
			if err != nil {
				return err
			}
			creds := cf.GetCredentialsStore(serverAddress)
			if serverAddress == name.DefaultRegistry {
				serverAddress = authn.DefaultAuthKey
			}
			if err := creds.Erase(serverAddress); err != nil {
				return err
			}

			if err := cf.Save(); err != nil {
				return err
			}
			log.Printf("logged out via %s", cf.Filename)
			return nil
		},
	}
	return cmd
}
