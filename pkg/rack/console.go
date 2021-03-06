package rack

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/convox/convox/pkg/common"
	"github.com/convox/convox/pkg/console"
	"github.com/convox/convox/sdk"
	"github.com/convox/stdcli"
)

type Console struct {
	ctx      *stdcli.Context
	host     string
	name     string
	provider string
	status   string
}

func CreateConsole(c *stdcli.Context, name string, md *Metadata) (*Console, error) {
	host, err := currentConsole(c)
	if err != nil {
		return nil, err
	}

	cc, err := consoleClient(c, host, "")
	if err != nil {
		return nil, err
	}

	r, err := cc.RackCreate(name, md.Provider, md.State, md.Vars)
	if err != nil {
		return nil, err
	}

	cr := &Console{ctx: c, name: name, provider: r.Provider, status: r.Status}

	return cr, nil
}

func InstallConsole(c *stdcli.Context, name, provider, version string, options map[string]string) error {
	return fmt.Errorf("console install not yet supported")
}

func LoadConsole(c *stdcli.Context, name string) (*Console, error) {
	rs, err := listConsole(c)
	if err != nil {
		return nil, err
	}

	for _, r := range rs {
		if r.Name() == name {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("no such console rack: %s", name)
}

func (c Console) Client() (sdk.Interface, error) {
	return c.client()
}

func (c Console) MarshalJSON() ([]byte, error) {
	h := map[string]string{
		"name": c.name,
		"type": "console",
	}

	return json.Marshal(h)
}

func (c Console) Name() string {
	return c.name
}

func (c Console) Parameters() (map[string]string, error) {
	cc, err := c.Client()
	if err != nil {
		return nil, err
	}

	s, err := cc.SystemGet()
	if err != nil {
		return nil, err
	}

	return s.Parameters, nil
}

func (c Console) Provider() string {
	return c.provider
}

func (c Console) Remote() bool {
	return true
}

func (c Console) Status() string {
	return c.status
}

func (c Console) Uninstall() error {
	return fmt.Errorf("console uninstall not yet supported")
}

func (c Console) UpdateParams(params map[string]string) error {
	if err := c.updateSupported(); err != nil {
		return err
	}

	d, err := c.direct()
	if err != nil {
		return err
	}

	if err := d.UpdateParams(params); err != nil {
		return err
	}

	return nil
}

func (c Console) UpdateVersion(version string) error {
	if err := c.updateSupported(); err != nil {
		return err
	}

	d, err := c.direct()
	if err != nil {
		return err
	}

	if err := d.UpdateVersion(version); err != nil {
		return err
	}

	return nil
}

func (c Console) client() (*console.Client, error) {
	cc, err := consoleClient(c.ctx, c.host, c.name)
	if err != nil {
		return nil, err
	}

	return cc, nil
}

func (c Console) direct() (*Direct, error) {
	cc, err := c.client()
	if err != nil {
		return nil, err
	}

	d, err := LoadDirect(cc)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (c Console) updateSupported() error {
	cc, err := c.client()
	if err != nil {
		return err
	}

	s, err := cc.SystemGet()
	if err != nil {
		return err
	}

	if !strings.HasPrefix(s.Version, "20") {
		return fmt.Errorf("console update of version 3 racks not yet supported")
	}

	return nil
}

func consoleClient(c *stdcli.Context, host, rack string) (*console.Client, error) {
	pw, err := currentPassword(c, host)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("https://convox:%s@%s", url.QueryEscape(pw), host)

	cc, err := console.NewClient(endpoint, rack, c)
	if err != nil {
		return nil, err
	}

	return cc, nil
}

func currentConsole(c *stdcli.Context) (string, error) {
	if h := os.Getenv("CONVOX_HOST"); h != "" {
		return h, nil
	}

	if h, _ := c.SettingRead("console"); h != "" {
		return h, nil
	}

	return "", nil
}

func currentPassword(c *stdcli.Context, host string) (string, error) {
	if pw := os.Getenv("CONVOX_PASSWORD"); pw != "" {
		return pw, nil
	}

	return c.SettingReadKey("auth", host)
}

func listConsole(c *stdcli.Context) ([]Console, error) {
	cs := []Console{}

	host, err := currentConsole(c)
	if err != nil {
		return nil, err
	}
	if host == "" {
		return []Console{}, nil
	}

	cc, err := consoleClient(c, host, "")
	if err != nil {
		return nil, err
	}

	rs, err := cc.RackList()
	if err != nil {
		return nil, err
	}

	for _, r := range rs {
		cs = append(cs, Console{
			ctx:      c,
			host:     host,
			name:     fmt.Sprintf("%s/%s", r.Organization.Name, r.Name),
			provider: common.CoalesceString(r.Provider, "unknown"),
			status:   r.Status,
		})
	}

	return cs, nil
}
