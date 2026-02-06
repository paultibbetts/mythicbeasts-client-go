/*
Package mythicbeasts-client-go works with the Mythic Beasts Raspberry Pi, VPS, and Proxy provisioning APIs.

See https://www.mythic-beasts.com/support/api for more on the Mythic Beasts APIs.

An API key is required for authentication. One can be obtained from https://www.mythic-beasts.com/customer/api-users.

	c, err := mythicbeasts.NewClient("API_KEY_ID", "API_KEY_SECRET")
	if err != nil {
		// handle error
	}

	ctx := context.Background()

	images, err := c.VPS().GetImages(ctx)
*/
package mythicbeasts
