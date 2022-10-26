package main

import (
    "flag"
    "fmt"
    "github.com/kevinburke/ssh_config"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "io"
)

func rewriteCfg(cfg *ssh_config.Config, targetPtn string, inputHostName string) (bool, error) {

    // try to rewrite config
    rewrited := false
    for _, host := range cfg.Hosts {
        // fmt.Println("patterns:", i, host.Patterns, host.Matches(targetPtn))

        // ホスト名マッチング
        isContainedWildCard := false
        if host.Matches(targetPtn) {
            // ワイルドカードが入っている場合は除外
            for _, pattern := range host.Patterns {
                if strings.Contains(pattern.String(), "*") {
                    isContainedWildCard = true
                    break
                }
            }
        }

        if !isContainedWildCard {
            for _, node := range host.Nodes {

                kv, ok := node.(*ssh_config.KV)

                // HostNameディレクティブが存在していた場合、値を修正
                if ok && kv.Key == "HostName" {
                    previousHostName := kv.Value
                    kv.Value = inputHostName
                    //kv.Comment = "This value was rewritten automatically"
                    fmt.Printf("Hostname rewrited: %s -> %s\n", previousHostName, kv.Value)

                    // 書き換え完了
                    rewrited = true
                    break
                }
            }

            // 全てのノードを確認してもHostNameが存在しない場合、新たにノードを書き加える
            if !rewrited {
                fmt.Println("adding Node...")
                newNode := &ssh_config.KV{
                    Key: "HostName", 
                    Value: inputHostName,
                    //Comment: "This value was added automatically",
                }
                host.Nodes = append(host.Nodes, newNode)
            }

        }

    }

    return rewrited, nil
}

func main() {

    // args parse
    targetPtnPtr := flag.String("target-ptn", "", "Target host pattern")
    inputHostNamePtr := flag.String("input-hostname", "", "Target host name")
    flag.Parse()

    targetPtn := *targetPtnPtr
    inputHostName := *inputHostNamePtr

    fmt.Println("Target hostname:", targetPtn)

    // define config path
    var confPath string

    if runtime.GOOS == "windows" {
        confPath = filepath.Join(os.Getenv("USERPROFILE"), ".ssh", "config")
    } else if runtime.GOOS == "linux" {
        confPath = filepath.Join(os.Getenv("HOME"), ".ssh", "config")
    } else {
        fmt.Println("unspported runtime: %s", runtime.GOOS)
        os.Exit(1)
    }

    // open target config file
    inputFs, err := os.Open(confPath)
    if err != nil {
        fmt.Printf("cannot open config file: %s\n", confPath)
        os.Exit(1)
    }

    // make backup file
    bkupFs, err := os.Create(confPath + ".old")
    if err != nil {
        fmt.Printf("cannot open backup file: %s\n", confPath + ".old")
        os.Exit(1)
    }
    defer bkupFs.Close()

    io.Copy(bkupFs, inputFs)

    // Decode target file
    fmt.Printf("[%s] Decoding config file path: %s\n", runtime.GOOS, confPath)
    inputFs.Seek(0, 0)
    cfg, _ := ssh_config.Decode(inputFs)

    inputFs.Close()

    // rewriting...
    rewrited, _ := rewriteCfg(cfg, targetPtn, inputHostName)

    // No Host exist?
    if !rewrited {
        //fmt.Println("Adding new host-pattern...")
        fmt.Println("No target-host exists. process interrupted.")
        os.Exit(1)
    }

    // overwrite target config file
    err = os.WriteFile(confPath, []byte(cfg.String()), 0664)
    if err != nil {
        fmt.Println(err)
    }

}
