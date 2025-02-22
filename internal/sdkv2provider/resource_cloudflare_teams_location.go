package sdkv2provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/cloudflare/cloudflare-go"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/consts"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudflareTeamsLocation() *schema.Resource {
	return &schema.Resource{
		Schema:        resourceCloudflareTeamsLocationSchema(),
		CreateContext: resourceCloudflareTeamsLocationCreate,
		ReadContext:   resourceCloudflareTeamsLocationRead,
		UpdateContext: resourceCloudflareTeamsLocationUpdate,
		DeleteContext: resourceCloudflareTeamsLocationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceCloudflareTeamsLocationImport,
		},
		Description: heredoc.Doc(`
			Provides a Cloudflare Teams Location resource. Teams Locations are
			referenced when creating secure web gateway policies.
		`),
		DeprecationMessage: "`cloudflare_teams_location` is now deprecated and will be removed in the next major version. Use `cloudflare_zero_trust_dns_location` instead.",
	}
}

func resourceCloudflareZeroTrustDNSLocation() *schema.Resource {
	return &schema.Resource{
		Schema:        resourceCloudflareTeamsLocationSchema(),
		CreateContext: resourceCloudflareTeamsLocationCreate,
		ReadContext:   resourceCloudflareTeamsLocationRead,
		UpdateContext: resourceCloudflareTeamsLocationUpdate,
		DeleteContext: resourceCloudflareTeamsLocationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceCloudflareTeamsLocationImport,
		},
		Description: heredoc.Doc(`
			Provides a Cloudflare Teams Location resource. Teams Locations are
			referenced when creating secure web gateway policies.
		`),
	}
}

func resourceCloudflareTeamsLocationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	accountID := d.Get(consts.AccountIDSchemaKey).(string)

	location, err := client.TeamsLocation(ctx, accountID, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "Location ID is invalid") {
			tflog.Info(ctx, fmt.Sprintf("Teams Location %s no longer exists", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("error finding Teams Location %q: %w", d.Id(), err))
	}

	if err := d.Set("name", location.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location name"))
	}
	if err := d.Set("networks", flattenTeamsLocationNetworks(location.Networks)); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location networks"))
	}

	if err := d.Set("ip", location.Ip); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location IP"))
	}
	if err := d.Set("doh_subdomain", location.Subdomain); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location DOH subdomain"))
	}
	if err := d.Set("anonymized_logs_enabled", location.AnonymizedLogsEnabled); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location anonimized log enablement"))
	}
	if err := d.Set("ipv4_destination", location.IPv4Destination); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location IPv4 destination"))
	}
	if err := d.Set("ipv4_destination_backup", location.IPv4DestinationBackup); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location IPv4 destination"))
	}
	if err := d.Set("client_default", location.ClientDefault); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location client default"))
	}
	if err := d.Set("ecs_support", location.ECSSupport); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location ecs support"))
	}
	if err := d.Set("dns_destination_ipv6_block_id", location.DNSDestinationIPv6BlockID); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location dns_destination_ipv6_block_id"))
	}

	if err := d.Set("dns_destination_ips_id", location.DNSDestinationIPsID); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location dns_destination_ipv6_block_id"))
	}

	if err := d.Set("endpoints", flattenTeamsEndpoints(location.Endpoints)); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing Location endpoints"))
	}

	return nil
}
func resourceCloudflareTeamsLocationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)

	accountID := d.Get(consts.AccountIDSchemaKey).(string)
	networks, err := inflateTeamsLocationNetworks(d.Get("networks"))
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Teams Location for account %q: %w, %v", accountID, err, networks))
	}
	newTeamLocation := cloudflare.TeamsLocation{
		Name:          d.Get("name").(string),
		Networks:      networks,
		ClientDefault: d.Get("client_default").(bool),
		ECSSupport:    cloudflare.BoolPtr(d.Get("ecs_support").(bool)),
	}

	endpoints, err := inflateTeamsLocationEndpoint(d.Get("endpoints"))
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Teams Location endpoints for account %q: %w, %v", accountID, err, networks))
	} else if endpoints != nil {
		newTeamLocation.Endpoints = endpoints
	}

	destinationIpId, ok := d.Get("dns_destination_ips_id").(string)
	if ok && destinationIpId != "" {
		newTeamLocation.DNSDestinationIPsID = &destinationIpId
	}
	destinationIpv6Id, ok := d.Get("dns_destination_ipv6_block_id").(string)
	if ok && destinationIpv6Id != "" {
		newTeamLocation.DNSDestinationIPv6BlockID = &destinationIpv6Id
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating Cloudflare Teams Location from struct: %+v", newTeamLocation))

	location, err := client.CreateTeamsLocation(ctx, accountID, newTeamLocation)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Teams Location for account %q: %w, %v", accountID, err, networks))
	}

	d.SetId(location.ID)
	return resourceCloudflareTeamsLocationRead(ctx, d, meta)
}
func resourceCloudflareTeamsLocationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	accountID := d.Get(consts.AccountIDSchemaKey).(string)
	networks, err := inflateTeamsLocationNetworks(d.Get("networks"))
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating Teams Location for account %q: %w, %v", accountID, err, networks))
	}
	updatedTeamsLocation := cloudflare.TeamsLocation{
		ID:            d.Id(),
		Name:          d.Get("name").(string),
		ClientDefault: d.Get("client_default").(bool),
		ECSSupport:    cloudflare.BoolPtr(d.Get("ecs_support").(bool)),
		Networks:      networks,
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Cloudflare Teams Location from struct: %+v", updatedTeamsLocation))

	teamsLocation, err := client.UpdateTeamsLocation(ctx, accountID, updatedTeamsLocation)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating Teams Location for account %q: %w", accountID, err))
	}
	if teamsLocation.ID == "" {
		return diag.FromErr(fmt.Errorf("failed to find Teams Location ID in update response; resource was empty"))
	}
	return resourceCloudflareTeamsLocationRead(ctx, d, meta)
}

func resourceCloudflareTeamsLocationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	id := d.Id()
	accountID := d.Get(consts.AccountIDSchemaKey).(string)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Cloudflare Teams Location using ID: %s", id))

	err := client.DeleteTeamsLocation(ctx, accountID, id)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting Teams Location for account %q: %w", accountID, err))
	}

	return resourceCloudflareTeamsLocationRead(ctx, d, meta)
}

func resourceCloudflareTeamsLocationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	attributes := strings.SplitN(d.Id(), "/", 2)

	if len(attributes) != 2 {
		return nil, fmt.Errorf("invalid id (\"%s\") specified, should be in format \"accountID/teamsLocationID\"", d.Id())
	}

	accountID, teamsLocationID := attributes[0], attributes[1]

	tflog.Debug(ctx, fmt.Sprintf("Importing Cloudflare Teams Location: id %s for account %s", teamsLocationID, accountID))

	d.Set(consts.AccountIDSchemaKey, accountID)
	d.SetId(teamsLocationID)

	resourceCloudflareTeamsLocationRead(ctx, d, meta)

	return []*schema.ResourceData{d}, nil
}

func inflateTeamsLocationNetworks(networks interface{}) ([]cloudflare.TeamsLocationNetwork, error) {
	var networkStructs []cloudflare.TeamsLocationNetwork
	if networks != nil {
		networkSet, ok := networks.(*schema.Set)
		if !ok {
			return nil, fmt.Errorf("error parsing network list")
		}
		for _, i := range networkSet.List() {
			network, ok := i.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("error parsing network")
			}
			networkStructs = append(networkStructs, cloudflare.TeamsLocationNetwork{
				Network: network["network"].(string),
			})
		}
	}
	return networkStructs, nil
}

func inflateTeamsLocationNetworksFromList(networks interface{}) ([]cloudflare.TeamsLocationNetwork, error) {
	var networkStructs []cloudflare.TeamsLocationNetwork
	if networks != nil {
		networkList, ok := networks.([]interface{})
		if !ok {
			return nil, fmt.Errorf("error parsing network list")
		}
		for _, i := range networkList {
			network, ok := i.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("error parsing network")
			}
			networkStructs = append(networkStructs, cloudflare.TeamsLocationNetwork{
				Network: network["network"].(string),
			})
		}
	}
	return networkStructs, nil
}

func inflateTeamsLocationEndpoint(endpoint interface{}) (*cloudflare.TeamsLocationEndpoints, error) {
	if endpoint == nil {
		return nil, nil
	}

	epList, ok := endpoint.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing endpoint list")
	}
	for _, i := range epList {
		epItem, ok := i.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("error parsing endpoint")
		}
		ipv4Endpoint, err := inflateIpv4Endpoint(epItem["ipv4"])
		if err != nil {
			return nil, fmt.Errorf("error parsing ipv4 endpoint")
		}

		ipv6Endpoint, err := inflateIpv6Endpoint(epItem["ipv6"])
		if err != nil {
			return nil, fmt.Errorf("error parsing ipv6 endpoint")
		}

		dotEndpoint, err := inflateDoTEndpoint(epItem["dot"])
		if err != nil {
			return nil, fmt.Errorf("error parsing dot endpoint")
		}

		dohEndpoint, err := inflateDohEndpoint(epItem["doh"])
		if err != nil {
			return nil, fmt.Errorf("error parsing doh endpoint")
		}

		return &cloudflare.TeamsLocationEndpoints{
			IPv4Endpoint: *ipv4Endpoint,
			IPv6Endpoint: *ipv6Endpoint,
			DotEndpoint:  *dotEndpoint,
			DohEndpoint:  *dohEndpoint,
		}, nil
	}
	return nil, fmt.Errorf("empty endpoint")
}

func inflateIpv4Endpoint(item interface{}) (*cloudflare.TeamsLocationIPv4EndpointFields, error) {
	epItems, ok := item.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing endpoint item")
	}

	return &cloudflare.TeamsLocationIPv4EndpointFields{
		Enabled: firstItemInSet(epItems)["enabled"].(bool),
	}, nil
}

func firstItemInSet(l []interface{}) map[string]interface{} {
	return l[0].(map[string]interface{})
}

func inflateIpv6Endpoint(item interface{}) (*cloudflare.TeamsLocationIPv6EndpointFields, error) {
	epItems, ok := item.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing endpoint item")
	}

	epItem := firstItemInSet(epItems)

	networks, err := inflateTeamsLocationNetworksFromList(epItem["networks"])
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint ipv6 networks")
	}
	return &cloudflare.TeamsLocationIPv6EndpointFields{
		TeamsLocationEndpointFields: cloudflare.TeamsLocationEndpointFields{
			Enabled:  epItem["enabled"].(bool),
			Networks: networks,
		},
	}, nil
}

func inflateDoTEndpoint(item interface{}) (*cloudflare.TeamsLocationDotEndpointFields, error) {
	epItems, ok := item.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing endpoint item")
	}

	epItem := firstItemInSet(epItems)
	networks, err := inflateTeamsLocationNetworksFromList(epItem["networks"])
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint dot networks")
	}
	return &cloudflare.TeamsLocationDotEndpointFields{
		RequireToken: epItem["require_token"].(bool),
		TeamsLocationEndpointFields: cloudflare.TeamsLocationEndpointFields{
			Enabled:  epItem["enabled"].(bool),
			Networks: networks,
		},
	}, nil
}

func inflateDohEndpoint(item interface{}) (*cloudflare.TeamsLocationDohEndpointFields, error) {
	epItems, ok := item.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing endpoint item")
	}

	epItem := firstItemInSet(epItems)

	networks, err := inflateTeamsLocationNetworksFromList(epItem["networks"])
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint dot networks")
	}
	return &cloudflare.TeamsLocationDohEndpointFields{
		RequireToken: epItem["require_token"].(bool),
		TeamsLocationEndpointFields: cloudflare.TeamsLocationEndpointFields{
			Enabled:  epItem["enabled"].(bool),
			Networks: networks,
		},
	}, nil
}

func flattenTeamsLocationNetworks(networks []cloudflare.TeamsLocationNetwork) []interface{} {
	var flattenedNetworks []interface{}
	for _, net := range networks {
		flattenedNetworks = append(flattenedNetworks, map[string]interface{}{
			"network": net.Network,
		})
	}
	return flattenedNetworks
}

func flattenTeamsLocationNetworksIntoList(networks []cloudflare.TeamsLocationNetwork) []interface{} {
	var flattenedNetworks []interface{}
	for _, net := range networks {
		flattenedNetworks = append(flattenedNetworks, map[string]interface{}{
			"network": net.Network,
		})
	}
	return flattenedNetworks
}

func flattenTeamsEndpoints(endpoint *cloudflare.TeamsLocationEndpoints) []interface{} {
	flattenedEndpoints := map[string]interface{}{
		"ipv4": flattenTeamsEndpointIpv4Field(endpoint.IPv4Endpoint),
		"ipv6": flattenTeamsEndpointIpv6Field(endpoint.IPv6Endpoint),
		"doh":  flattenTeamsEndpointDOHField(endpoint.DohEndpoint),
		"dot":  flattenTeamsEndpointDOTField(endpoint.DotEndpoint),
	}
	return []interface{}{flattenedEndpoints}
}

func flattenTeamsEndpointIpv4Field(field cloudflare.TeamsLocationIPv4EndpointFields) []map[string]interface{} {
	return []map[string]interface{}{{
		"enabled":                field.Enabled,
		"authentication_enabled": field.AuthenticationEnabled,
	}}
}

func flattenTeamsEndpointIpv6Field(field cloudflare.TeamsLocationIPv6EndpointFields) []map[string]interface{} {
	return []map[string]interface{}{{
		"enabled":                field.Enabled,
		"authentication_enabled": field.AuthenticationEnabledUIHelper,
		"networks":               flattenTeamsLocationNetworksIntoList(field.Networks),
	}}
}

func flattenTeamsEndpointDOTField(field cloudflare.TeamsLocationDotEndpointFields) []map[string]interface{} {
	return []map[string]interface{}{{
		"require_token":          field.RequireToken,
		"enabled":                field.Enabled,
		"authentication_enabled": field.AuthenticationEnabledUIHelper,
		"networks":               flattenTeamsLocationNetworksIntoList(field.Networks),
	}}
}

func flattenTeamsEndpointDOHField(field cloudflare.TeamsLocationDohEndpointFields) []map[string]interface{} {
	return []map[string]interface{}{{
		"require_token":          field.RequireToken,
		"enabled":                field.Enabled,
		"authentication_enabled": field.AuthenticationEnabledUIHelper,
		"networks":               flattenTeamsLocationNetworksIntoList(field.Networks),
	}}
}
