package client

// Client a Cabourotte client
type Client struct {
	Config *Configuration
}

// New creates a new client
func New(config *Configuration) Client {
	client := Client{
		Config: config,
	}
	return client
}

func (c *Client) request() {

}

func (c *Client) List() {

}
