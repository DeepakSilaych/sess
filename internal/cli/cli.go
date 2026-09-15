package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DeepakSilaych/sess/internal/api"
	"github.com/DeepakSilaych/sess/internal/backend"
	"github.com/DeepakSilaych/sess/internal/provision"
	"github.com/DeepakSilaych/sess/internal/store"
	"github.com/DeepakSilaych/sess/internal/transport"
	"github.com/DeepakSilaych/sess/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"io"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"
)

func New(out, errout io.Writer) *cobra.Command {
	var host string
	root := &cobra.Command{Use: "sess", Short: "Persistent SSH terminals, powered by zmx", Long: "sess keeps named terminals running on your VM.\nCreate a session, detach, and return to the same shell later.", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errout)
	root.PersistentFlags().StringVarP(&host, "host", "h", "", "SSH destination or alias (overrides the saved default)")
	root.PersistentFlags().Bool("help", false, "Show help")
	root.SetHelpCommand(&cobra.Command{Use: "help [command]", Short: "Show help for a command", RunE: func(c *cobra.Command, args []string) error {
		target, _, e := root.Find(args)
		if e != nil {
			return e
		}
		return target.Help()
	}})
	client := func() (*transport.Client, error) {
		h, e := store.Resolve(host)
		if e != nil {
			return nil, e
		}
		return transport.New(h)
	}
	interactive := func() bool { return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) }
	attach := func(ctx context.Context, c *transport.Client, s api.Session) error {
		fmt.Fprintf(errout, "Attaching to %s / %s · Ctrl+\\ to detach\n", c.Host, s.Name)
		e := c.Attach(ctx, s, errout)
		if e == nil {
			fmt.Fprintf(errout, "\nConnection ended · %s / %s\n", c.Host, s.Name)
		}
		return e
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("unknown command %q; run sess --help", args[0])
		}
		c, e := client()
		if e != nil {
			return e
		}
		if !interactive() {
			ss, e := c.List(cmd.Context())
			if e != nil {
				return e
			}
			printList(out, c.Host, ss, false, false)
			return nil
		}
		if os.Getenv("ZMX_SESSION") != "" {
			return fmt.Errorf("detach from your current persistent terminal before opening another attachment")
		}
		lastError := ""
		for {
			choice, e := tui.Run(cmd.Context(), c.Host, lastError)
			if e != nil {
				return e
			}
			if choice.Session == nil {
				return nil
			}
			c, _ = transport.New(choice.Host)
			lastError = ""
			if e = attach(cmd.Context(), c, *choice.Session); e != nil {
				if cmd.Context().Err() != nil {
					return e
				}
				lastError = e.Error()
			}
		}
	}
	root.AddCommand(&cobra.Command{Use: "init <host>", Short: "Prepare sess and zmx on a VM", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if host != "" {
			return fmt.Errorf("init takes its host as an argument; use sess init %s", host)
		}
		c, e := transport.New(args[0])
		if e != nil {
			return e
		}
		return provision.Init(cmd.Context(), c, out)
	}})
	root.AddCommand(&cobra.Command{Use: "set --host <host>", Short: "Save the default SSH host", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if host == "" {
			return fmt.Errorf("provide a destination: sess set --host <host>")
		}
		if e := store.SetHost(host); e != nil {
			return e
		}
		fmt.Fprintf(out, "Default host: %s\n", host)
		return nil
	}})
	var detached bool
	newcmd := &cobra.Command{Use: "new <name>", Aliases: []string{"n"}, Short: "Create a persistent terminal and attach", Args: cobra.ExactArgs(1), Example: "  sess new api\n  sess n tests -h dev\n  sess new build --detach", RunE: func(cmd *cobra.Command, args []string) error {
		if e := api.ValidateName(args[0]); e != nil {
			return e
		}
		if !detached && !interactive() {
			return fmt.Errorf("attachment requires an interactive terminal; use sess new %s --detach", args[0])
		}
		if !detached && os.Getenv("ZMX_SESSION") != "" {
			return fmt.Errorf("detach from your current persistent terminal first")
		}
		c, e := client()
		if e != nil {
			return e
		}
		s, e := c.Create(cmd.Context(), args[0])
		if e != nil {
			return e
		}
		fmt.Fprintf(out, "Created %s / %s\n", c.Host, s.Name)
		if detached {
			return nil
		}
		return attach(cmd.Context(), c, s)
	}}
	newcmd.Flags().BoolVarP(&detached, "detach", "d", false, "Create without attaching (for scripts)")
	root.AddCommand(newcmd)
	root.AddCommand(&cobra.Command{Use: "attach <name>", Aliases: []string{"a"}, Short: "Attach to an existing terminal", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if e := api.ValidateName(args[0]); e != nil {
			return e
		}
		if !interactive() {
			return fmt.Errorf("attach requires an interactive terminal")
		}
		if os.Getenv("ZMX_SESSION") != "" {
			return fmt.Errorf("detach from your current persistent terminal first")
		}
		c, e := client()
		if e != nil {
			return e
		}
		s, e := c.Find(cmd.Context(), args[0])
		if e != nil {
			return e
		}
		return attach(cmd.Context(), c, s)
	}})
	root.AddCommand(&cobra.Command{Use: "remove <name>", Aliases: []string{"rm"}, Short: "Terminate a session and its programs", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if e := api.ValidateName(args[0]); e != nil {
			return e
		}
		c, e := client()
		if e != nil {
			return e
		}
		s, e := c.Find(cmd.Context(), args[0])
		if e != nil {
			return e
		}
		fmt.Fprintf(errout, "Removing %s / %s\n", c.Host, s.Name)
		if e = c.Remove(cmd.Context(), s); e != nil {
			return e
		}
		fmt.Fprintf(out, "Removed %s\n", s.Name)
		return nil
	}})
	var jsonOut, quiet bool
	ls := &cobra.Command{Use: "ls", Short: "List live sessions on the selected VM", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if jsonOut && quiet {
			return fmt.Errorf("choose either --json or --quiet")
		}
		c, e := client()
		if e != nil {
			return e
		}
		ss, e := c.List(cmd.Context())
		if e != nil {
			return e
		}
		return printList(out, c.Host, ss, jsonOut, quiet)
	}}
	ls.Flags().BoolVar(&jsonOut, "json", false, "Print structured JSON")
	ls.Flags().BoolVarP(&quiet, "quiet", "q", false, "Print session names only")
	root.AddCommand(ls)
	root.AddCommand(&cobra.Command{Use: "detach", Short: "Inside a session, detach all of its clients", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if host != "" {
			return fmt.Errorf("detach runs inside the current session and accepts no host")
		}
		z, e := backend.Open()
		if e != nil {
			return e
		}
		return z.Detach(cmd.Context())
	}})
	root.AddCommand(&cobra.Command{Use: "doctor", Short: "Check SSH and remote session support", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, e := client()
		if e != nil {
			return e
		}
		fmt.Fprintf(out, "Host       %s\nClient     sess %s\n", c.Host, api.Version)
		r, e := c.Request(cmd.Context(), api.Request{Action: "ping"})
		if e != nil {
			return e
		}
		fmt.Fprintf(out, "Remote     sess %s\nBackend    %s\nProtocol   %d\n\nReady for persistent terminals.\n", r.Version, r.Backend, r.Protocol)
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "version", Short: "Show the installed version", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, args []string) { fmt.Fprintf(out, "sess %s\n", api.Version) }})
	completion := &cobra.Command{Use: "completion <bash|zsh|fish>", Short: "Generate shell completions", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return root.GenBashCompletionV2(out, true)
		case "zsh":
			return root.GenZshCompletion(out)
		case "fish":
			return root.GenFishCompletion(out, true)
		default:
			return fmt.Errorf("choose bash, zsh or fish")
		}
	}}
	root.AddCommand(completion)
	for _, cmd := range root.Commands() {
		if cmd.Name() == "attach" || cmd.Name() == "remove" {
			cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				if len(args) > 0 {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				c, e := client()
				if e != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Second)
				defer cancel()
				ss, e := c.List(ctx)
				if e != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				names := []string{}
				for _, s := range ss {
					if strings.HasPrefix(s.Name, toComplete) {
						names = append(names, s.Name)
					}
				}
				return names, cobra.ShellCompDirectiveNoFileComp
			}
		}
	}
	root.RegisterFlagCompletionFunc("host", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		c, _ := store.Load()
		return c.Hosts, cobra.ShellCompDirectiveNoFileComp
	})
	return root
}
func printList(out io.Writer, host string, ss []api.Session, jsonOut, quiet bool) error {
	if jsonOut {
		return json.NewEncoder(out).Encode(struct {
			Host     string        `json:"host"`
			Sessions []api.Session `json:"sessions"`
		}{host, ss})
	}
	if quiet {
		for _, s := range ss {
			fmt.Fprintln(out, s.Name)
		}
		return nil
	}
	fmt.Fprintf(out, "Host: %s\n\n", host)
	if len(ss) == 0 {
		fmt.Fprintln(out, "No sessions yet. Start one with: sess new <name>")
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 4, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tCLIENTS\tAGE\tDIRECTORY")
	for _, s := range ss {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", s.Name, s.Status, s.Clients, tui.Age(s.Created), tui.Clean(s.Directory))
	}
	return w.Flush()
}
func Main() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if e := New(os.Stdout, os.Stderr).ExecuteContext(ctx); e != nil {
		if errors.Is(e, context.Canceled) {
			fmt.Fprintln(os.Stderr, "\nDisconnected. Remote sessions are unchanged.")
			return 130
		}
		fmt.Fprintln(os.Stderr, "Error: "+e.Error())
		return 1
	}
	return 0
}
