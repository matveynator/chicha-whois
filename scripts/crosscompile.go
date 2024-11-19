package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

func main() {
    version := "1.0-001"
    executionFile := "ripe-country-list"

    // Get the root path of the Git repository
    gitRootPath, err := getGitRootPath()
    if err != nil {
        log.Fatalf("Error getting Git root path: %v", err)
    }

    // Step 2: Run tests on all modules
    fmt.Println("Performing tests on all modules...")
    if err := runCommand("go", "test", "./..."); err != nil {
        fmt.Println("Tests on all modules failed.")
        fmt.Println("Press Enter to continue compilation or CTRL+C to abort.")
        bufio.NewReader(os.Stdin).ReadBytes('\n')
    } else {
        fmt.Println("Tests on all modules passed.")
    }

    // Step 3: Set up directories
    binariesPath := filepath.Join(gitRootPath, "binaries", version)
    err = os.MkdirAll(binariesPath, os.ModePerm)
    if err != nil {
        log.Fatalf("Error creating binaries directory: %v", err)
    }

    latestLink := filepath.Join(gitRootPath, "binaries", "latest")
    os.Remove(latestLink)
    err = os.Symlink(version, latestLink)
    if err != nil {
        log.Printf("Warning: Failed to create symlink 'latest': %v", err)
    }

    // Step 4: Build for multiple OS and architectures
    osList := []string{
        "android", "aix", "darwin", "dragonfly", "freebsd",
        "illumos", "ios", "js", "linux", "netbsd",
        "openbsd", "plan9", "solaris", "windows", "zos",
    }

    archList := []string{
        "amd64", "386", "arm", "arm64", "mips64",
        "mips64le", "mips", "mipsle", "ppc64",
        "ppc64le", "riscv64", "s390x", "wasm",
    }

    for _, osName := range osList {
        for _, arch := range archList {
            targetOSName := osName
            execFileName := executionFile

            if osName == "windows" {
                execFileName += ".exe"
            } else if osName == "darwin" {
                targetOSName = "mac"
            }

            outputDir := filepath.Join(binariesPath, "no-gui", targetOSName, arch)
            err := os.MkdirAll(outputDir, os.ModePerm)
            if err != nil {
                log.Printf("Error creating output directory %s: %v", outputDir, err)
                continue
            }

            outputPath := filepath.Join(outputDir, execFileName)

            //fmt.Printf("Building for %s/%s...\n", osName, arch)
            buildCmd := exec.Command("go", "build", "-o", outputPath, "ripe-country-list.go")
            buildCmd.Env = append(os.Environ(),
                "GOOS="+osName,
                "GOARCH="+arch,
            )

            if err := buildCmd.Run(); err != nil {
				//fmt.Printf("Failed to build for %s/%s: %v\n", osName, arch, err)

				// Удаляем директорию
				err = os.RemoveAll(outputDir)
				if err != nil {
					log.Printf("Error removing output directory %s: %v", outputDir, err)
				}

                continue
			} else {

				err = os.Chmod(outputPath, 0755)
				if err != nil {
					log.Printf("Error setting permissions on %s: %v", outputPath, err)
				}

				fmt.Printf("Successfully built %s for %s/%s\n", execFileName, osName, arch)
			}
        }
    }

    // Step 5: Optional deployment over SSH
    fmt.Println("Do you want to deploy the binaries over SSH? (y/n)")
    var response string
    fmt.Scanln(&response)
    if strings.ToLower(response) == "y" {
        deployPath := "/home/files/public_html/ripe-country-list/"
        remoteHost := "files@files.zabiyaka.net"

        err = runCommand("rsync", "-avP", binariesPath+"/", fmt.Sprintf("%s:%s", remoteHost, deployPath))
        if err != nil {
            log.Printf("Error deploying binaries: %v", err)
        } else {
            fmt.Println("Deployment completed successfully.")
        }
    } else {
        fmt.Println("Deployment skipped.")
    }
}

// Helper function to run a command
func runCommand(name string, args ...string) error {
    cmd := exec.Command(name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

// Helper function to get the Git root path
func getGitRootPath() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--show-toplevel")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

