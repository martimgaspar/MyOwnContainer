//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	//"strconv" //Not needed for rootless?
	"syscall"
)

var cgroups = "/sys/fs/cgroup"
var custom_cgroup = filepath.Join(cgroups, "liz")

// docker           run image <cmd> <params>
// go run main.go   run       <cmd> <params>

func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("bad command")
	}
}

func run() {
	fmt.Printf("Running %v as %d\n", os.Args[2:], os.Getpid())

	cmd := exec.Command("proc/self/exe", append([]string{"child"}, os.Args[2:]...)...) // Eventually the goal is to direct the input and output from my own application!
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
		UidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      1000,
			Size:        1,
		}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      1000,
			Size:        1,
		}},
	}
	must(cmd.Run())
}

func child() {
	fmt.Printf("Running %v as %d\n", os.Args[2:], os.Getpid())

	//cg() Dont need for rootless

	syscall.Sethostname([]byte("container"))
	must(syscall.Chroot("/vagrant/ubuntu-fs")) //Replace this with whatever root you have with files that should be accesible by container's own root
	must(syscall.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))

	cmd := exec.Command(os.Args[2], os.Args[3:]...) //Eventually the goal is to direct the input and output from my own application!
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	must(cmd.Run())

	must(syscall.Unmount("/proc", 0))
}

/*
func cg() { // Control group to limit processes container can use
	os.Mkdir(custom_cgroup, 0755)

	must(os.WriteFile(filepath.Join(custom_cgroup, "pids.max"), []byte("20"), 0644))
	must(os.WriteFile(filepath.Join(custom_cgroup, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0644))
} */ // Dont need for rootless

func cgCleanup() error {
	alive, err := os.ReadFile(filepath.Join(custom_cgroup, "pids.current"))
	if err != nil { // or must(err).. but then it'll look weird..
		panic(err)
	}

	if alive[0] != uint8(48) {
		must(os.WriteFile(filepath.Join(custom_cgroup, "cgroup.kill"), []byte("1"), 0644))
	}
	must(os.Remove(custom_cgroup))

	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
