package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type gitlab struct {
	serverURL      string
	projectsConfig string
}

type projects []project
type project struct {
	Name         string `yaml:"name"`
	ID           int    `yaml:"id"`
	Token        string `yaml:"token"`
	Dependencies []dependency
}

type dependency struct {
	GroupID    string `yaml:"groupId"`
	ArtifactID string `yaml:"artifactId"`
}

func (g gitlab) notify(a artifact) error {

	log.Printf("notify gitlab, artifact: %v", a)

	ps, err := loadProjects(g.projectsConfig)
	if err != nil {
		return fmt.Errorf("failed to load projects: %v", err)
	}

	// [improvement] create reverse index of artifact & group -> project
	for _, p := range ps {
		for _, d := range p.Dependencies {
			if d.ArtifactID == a.Component.Name && d.GroupID == a.Component.Group {
				log.Printf("trigger pipeline %s %d", p.Name, p.ID)
				err = g.triggerPipeline(p.ID, p.Token)
				if err != nil {
					log.Fatalf("failed to trigger pipeline of %s: %v", p.Name, err)
				}
			}
		}
	}

	return nil
}

func loadProjects(path string) (projects, error) {
	// [improvement] use https://github.com/fsnotify/fsnotify/blob/master/example_test.go and keep config in memory
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ps := projects{}
	err = yaml.Unmarshal(dat, &ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// triggers pipeline following https://docs.gitlab.com/ce/ci/triggers/README.html#triggering-a-pipeline
func (g gitlab) triggerPipeline(id int, t string) error {

	c := http.Client{Timeout: time.Second * 10}
	u := g.serverURL + "/api/v4/projects/" + strconv.Itoa(id) + "/trigger/pipeline"
	resp, err := c.PostForm(u, url.Values{"token": {t}, "ref": {"master"}})
	if err != nil {
		return fmt.Errorf("failed post to project %v: %v", id, err)
	}
	err = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to close response: %v", err)
	}
	return nil
}
