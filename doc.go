/*
Package mythicbeasts-client-go works with the Mythic Beasts Raspberry Pi and VPS provisioning APIs.

See https://www.mythic-beasts.com/support/api for more on the Mythic Beasts APIs.

An API key is required for authentication. One can be obtained from https://www.mythic-beasts.com/customer/api-users.

To authenticate you must construct a new client using your API Key ID and secret:

	c := mythicbeasts.NewClient("YOUR_API_KEYID", "YOUR_API_SECRET")

Now you are able to interact with the Raspberry Pi (Pi) and VPS resources:

	operatingSystems, err := c.GetPiOperatingSystems()
	images, err := c.GetVPSImages()
*/
package mythicbeasts
