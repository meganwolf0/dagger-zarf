package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

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

	zarf := client.Container().From("ttl.sh/zarf-20230804:8h")

	// Build Packages
	// Flux
	fluxCache := client.CacheVolume("flux")
	buildPackage(client, ctx, zarf, "flux", fluxCache)

	// Test

	// Publish

	return nil
}

func buildPackage(client *dagger.Client, ctx context.Context, c *dagger.Container, pkg string, pkgCache *dagger.CacheVolume) {
	// Build Zarf package
	pkgDir := client.Directory().WithDirectory(pkg, client.Host().Directory(fmt.Sprintf("packages/%s", pkg)))
	zarfCmd := fmt.Sprintf("zarf package create ./pkgs/%s --confirm --no-progress --tmpdir /tmp/zarf -o ./build 2>/dev/null", pkg)

	build := c.Pipeline(pkg).
		WithMountedCache("/app/pkgs", pkgCache, dagger.ContainerWithMountedCacheOpts{Source: pkgDir}).
		WithWorkdir("/app").
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
