package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/althk/drivefs/driveapi"
	"github.com/althk/drivefs/fusehooks"
	"google.golang.org/api/drive/v3"
)

var (
	mountPath       = flag.String("mntpoint", "", "Mount dir for GDrive")
	credentialsPath = flag.String("credsfile", "", "Path to creds json file")
	tokenPath       = flag.String("tokenfile", "", "Path to oauth token")
)
var svc *drive.Service

func main() {
	flag.Parse()
	if flag.NFlag() != 3 {
		flag.Usage()
		os.Exit(2)
	}

	// Handle program interruption (signint) so that we can
	// cleanly unmount FUSE fs before exiting.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	b, err := os.ReadFile(*credentialsPath)
	if err != nil {
		log.Fatalf("Unable to read credentials.json: %v", err)
	}
	svc = driveapi.InitWithConfigJSON(ctx, b, *tokenPath)
	fmt.Println("Drive client initialized")

	if err := mount(ctx, stop, *mountPath); err != nil {
		log.Fatalf("Mount err: %v\n", err)
	}
}

func mount(ctx context.Context, stop context.CancelFunc, mnt string) error {
	c, err := fuse.Mount(mnt)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done() // Program interrupted
		_ = fuse.Unmount(mnt)
		_ = c.Close()
		fmt.Println("Program interrupted")
		stop()

	}()

	// Unmount first, then close the FUSE conn.
	defer fuse.Unmount(mnt)
	defer c.Close()

	dfs := &fusehooks.FS{Ctx: ctx, DriveSvc: svc}
	if err := fs.Serve(c, dfs); err != nil {
		_ = fuse.Unmount(mnt)
		fmt.Printf("serving ended: %v", err)
	}
	return nil
}
