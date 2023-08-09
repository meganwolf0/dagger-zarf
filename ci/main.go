package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"dagger.io/dagger"
)

type BuildOpts struct {
	registry1 bool
	ecr       bool
}

func main() {
	if err := build(context.Background()); err != nil {
		fmt.Println(err)
	}
}

func build(ctx context.Context) error {
	fmt.Println("Building with Dagger")

	// Initialize Dagger client
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	// Set secrets
	registry1Username := client.SetSecret("registry1Username", os.Getenv("REGISTRY1_USERNAME"))
	registry1Password := client.SetSecret("registry1Password", os.Getenv("REGISTRY1_PASSWORD"))

	// Get Zarf container with registry login
	zarf := client.Container().From("ghcr.io/meganwolf0/zarf:v0.28.4").
		WithSecretVariable("REGISTRY1_USERNAME", registry1Username).
		WithSecretVariable("REGISTRY1_PASSWORD", registry1Password)

	// Build Packages - utilize parallelization and wait groups
	var wg sync.WaitGroup

	// pkgs := []string{"flux", "podinfo"}
	// for _, pkg := range pkgs {
	// 	wg.Add(1)
	// 	go buildPackage(client, ctx, zarf, pkg, &wg)
	// }

	wg.Add(2)
	go buildPackage(client, ctx, zarf, "flux", &wg, BuildOpts{registry1: true})
	go buildPackage(client, ctx, zarf, "podinfo", &wg)

	// WHY when I include this section, it re-runs everything?? Even if I don't include mounted cache
	// bbCache := client.CacheVolume("bb-cache") // since this is a big package, try caching
	// zarf = zarf.WithMountedCache("~/.zarf-cache", bbCache)
	// wg.Add(1)
	// go buildPackage(client, ctx, zarf, "bigbang", &wg, BuildOpts{registry1: true})

	// Try with version of big bang that doesn't use the zarf "extension"
	wg.Add(1)
	go buildPackage(client, ctx, zarf, "big-bang-core", &wg, BuildOpts{registry1: true})

	wg.Wait()

	// Test -> how to enforce testing on only packages that have changed?

	// Publish

	return nil
}

func buildPackage(client *dagger.Client, ctx context.Context, c *dagger.Container, pkg string, wg *sync.WaitGroup, opts ...BuildOpts) {
	defer wg.Done()

	// Build Zarf package
	pkgDir := client.Directory().WithDirectory(pkg, client.Host().Directory(fmt.Sprintf("packages/%s", pkg)))
	zarfCmd := ""
	sep := " && "

	for i := len(opts) - 1; i >= 0; i-- {
		// login to registry1
		if opts[i].registry1 {
			zarfCmd += "zarf tools registry login -u $REGISTRY1_USERNAME -p $REGISTRY1_PASSWORD registry1.dso.mil" + sep
		}

		// login to ECR - tbd
	}

	zarfCmd += fmt.Sprintf("zarf package create ./pkgs/%s --confirm --no-progress --tmpdir /tmp/zarf -o ./build 2>/dev/null", pkg)
	// zarfCmd += "mkdir build && echo $REGISTRY1_USERNAME > ./build/out.txt" // test variable passage
	// zarfCmd += "zarf tools registry pull registry1.dso.mil/ironbank/opensource/alpinelinux/alpine:3.18.2 alpine-3-18-2.tar.gz" // test registry1 login

	build := c.Pipeline(pkg).
		WithMountedDirectory("/app/pkgs", pkgDir).
		WithWorkdir("/app").
		WithEnvVariable("ZARF_CONFIG", fmt.Sprintf("./pkgs/%s/zarf-config.yaml", pkg)).
		WithExec([]string{
			"/bin/sh", "-c",
			zarfCmd,
		}).
		Directory("build")

	_, err := build.Export(ctx, "./build")
	if err != nil {
		panic(err)
	}

}
