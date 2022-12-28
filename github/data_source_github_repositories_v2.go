package github

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/google/go-github/v48/github"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceGithubRepositoriesV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGithubRepositoriesV2Read,

		Schema: map[string]*schema.Schema{
			"filters": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
			},
			"repositories": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"full_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"default_branch": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceGithubRepositoriesV2Read(d *schema.ResourceData, meta interface{}) error {
	var resourceId bytes.Buffer
	client := meta.(*Owner).v3client
	regexes := []*regexp.Regexp{}
	filters := d.Get("filters").([]interface{})

	for _, f := range filters {
		rx, err := regexp.Compile(f.(string))
		if err != nil {
			return fmt.Errorf("[ERROR] not able to compile regex '%s': %v", f, err)
		}
		regexes = append(regexes, rx)
		resourceId.WriteString(f.(string))
	}

	repos, err := listOrgRepositories(client, regexes)
	if err != nil {
		return err
	}

	d.SetId(resourceId.String())
	if err := d.Set("repositories", repos); err != nil {
		return fmt.Errorf("Not able to add repositories, %q", err)
	}

	return nil
}

func listOrgRepositories(client *github.Client, regexes []*regexp.Regexp) ([]map[string]interface{}, error) {
	repos := make([]map[string]interface{}, 0)

	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		listRepos, response, err := client.Repositories.ListByOrg(context.Background(), "wkda", opt)
		if err != nil {
			return nil, fmt.Errorf("[ERROR] Failed to list repositories: %q", err)
		}

		for _, r := range listRepos {
			if *r.Archived || *r.Disabled {
				continue
			}

			for _, regex := range regexes {
				if regex.MatchString(*r.Name) {
					log.Printf("[DEBUG] adding repository: %s", *r.FullName)
					repos = append(repos, map[string]interface{}{
						"name":           r.Name,
						"full_name":      r.FullName,
						"default_branch": r.DefaultBranch,
					})
				}
			}
		}

		opt.Page = response.NextPage
		if response.NextPage == 0 {
			break
		}
	}

	return repos, nil
}
