package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

type DockerContainer struct {
}

// based off of https://github.com/dotcloud/docker/blob/2a711d16e05b69328f2636f88f8eac035477f7e4/utils/utils.go
func parseHost(addr string) (string, string, error) {

	var (
		proto string
		host  string
		port  int
	)
	addr = strings.TrimSpace(addr)
	switch {
	case addr == "tcp://":
		return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
	case strings.HasPrefix(addr, "unix://"):
		proto = "unix"
		addr = strings.TrimPrefix(addr, "unix://")
		if addr == "" {
			addr = "/var/run/docker.sock"
		}
	case strings.HasPrefix(addr, "tcp://"):
		proto = "tcp"
		addr = strings.TrimPrefix(addr, "tcp://")
	case strings.HasPrefix(addr, "fd://"):
		return "fd", addr, nil
	case addr == "":
		proto = "unix"
		addr = "/var/run/docker.sock"
	default:
		if strings.Contains(addr, "://") {
			return "", "", fmt.Errorf("Invalid bind address protocol: %s", addr)
		}
		proto = "tcp"
	}

	if proto != "unix" && strings.Contains(addr, ":") {
		hostParts := strings.Split(addr, ":")
		if len(hostParts) != 2 {
			return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
		}
		if hostParts[0] != "" {
			host = hostParts[0]
		} else {
			host = "127.0.0.1"
		}

		if p, err := strconv.Atoi(hostParts[1]); err == nil && p != 0 {
			port = p
		} else {
			return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
		}

	} else if proto == "tcp" && !strings.Contains(addr, ":") {
		return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
	} else {
		host = addr
	}
	if proto == "unix" {
		return proto, host, nil

	}
	return proto, fmt.Sprintf("%s:%d", host, port), nil
}

func splitDockerImage(img string) (string, string, string) {
	index := 0
	repository := img
	var registry, tag string
	if strings.Contains(img, "/") {
		separator := strings.Index(img, "/")
		registry = img[index:separator]
		index = separator + 1
		repository = img[index:]
	}

	if strings.Contains(repository, ":") {
		separator := strings.Index(repository, ":")
		tag = repository[separator+1:]
		repository = repository[0:separator]
	}

	return registry, repository, tag
}

func getContainers(client *docker.Client) ([]*RuntimeContainer, error) {

	apiContainers, err := client.ListContainers(docker.ListContainersOptions{
		All:  false,
		Size: false,
	})
	if err != nil {
		return nil, err
	}

	containers := []*RuntimeContainer{}
	for _, apiContainer := range apiContainers {
		container, err := client.InspectContainer(apiContainer.ID)
		if err != nil {
			log.Printf("error inspecting container: %s: %s\n", apiContainer.ID, err)
			continue
		}

		registry, repository, tag := splitDockerImage(container.Config.Image)
		runtimeContainer := &RuntimeContainer{
			ID: container.ID,
			Image: DockerImage{
				Registry:   registry,
				Repository: repository,
				Tag:        tag,
			},
			Name:      strings.TrimLeft(container.Name, "/"),
			Hostname:  container.Config.Hostname,
			Gateway:   container.NetworkSettings.Gateway,
			Addresses: []Address{},
			Env:       make(map[string]string),
			Volumes:   make(map[string]Volume),
		}
		for k, v := range container.NetworkSettings.Ports {
			address := Address{
				IP:    container.NetworkSettings.IPAddress,
				Port:  k.Port(),
				Proto: k.Proto(),
			}
			if len(v) > 0 {
				address.HostPort = v[0].HostPort
			}
			runtimeContainer.Addresses = append(runtimeContainer.Addresses,
				address)

		}
		if len(runtimeContainer.Addresses) == 0 {
			fakeAddress := Address{
				IP: container.NetworkSettings.IPAddress,
			}
			runtimeContainer.Addresses = append(runtimeContainer.Addresses,
				fakeAddress)
		}

		for k, v := range container.Volumes {
			runtimeContainer.Volumes[k] = Volume{
				Path:      k,
				HostPath:  v,
				ReadWrite: container.VolumesRW[k],
			}
		}

		for _, entry := range container.Config.Env {
			parts := strings.Split(entry, "=")
			runtimeContainer.Env[parts[0]] = parts[1]
		}

		containers = append(containers, runtimeContainer)
	}
	return containers, nil

}
