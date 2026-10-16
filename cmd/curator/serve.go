package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aitr/31-course-help/internal/docs"
	"aitr/31-course-help/internal/gitx"
	"aitr/31-course-help/internal/web"
)

func init() {
	register("serve", "-addr 127.0.0.1:8781 [-root dir] [-data dir] [-llm cloud|local] [-git-init]  окно куратора (страница, /api/ask по SSE, /api/state)", runServe)
}

// runServe — `curator serve`: окно куратора. -git-init — если корень не
// репозиторий, сделать его таковым (копия каталога в сценарии записи:
// git_branch и свежесть индекса должны работать и на копии).
func runServe(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8781", "адрес окна")
	rootFlag := fs.String("root", "", "корень курса (по умолчанию — вверх от текущего каталога)")
	dataFlag := fs.String("data", "", "каталог данных с course.db (по умолчанию <корень>/tools/curator/data, <корень>/data или $CURATOR_DATA)")
	llmFlag := fs.String("llm", "", "провайдер модели: cloud (DeepSeek) или local; пусто — $LLM_PROVIDER или cloud")
	featFlag := fs.String("features", "", "механизмы: «-help.rag», «-help.tools», «all», «default»")
	indexFlag := fs.String("index", "", "индекс базы (по умолчанию structure)")
	gitInit := fs.Bool("git-init", false, "сделать корень git-репозиторием (ветка main, один коммит), если он им не является")
	if err := fs.Parse(args); err != nil {
		return err
	}
	loadDotEnv(*rootFlag)
	root, err := openRoot(*rootFlag)
	if err != nil {
		return err
	}
	if *gitInit {
		if err := ensureGit(ctx, root.Dir); err != nil {
			return err
		}
	}
	cl, model, err := curatorLLM(*llmFlag)
	if err != nil {
		// Окно поднимается и без модели: страница и состояние работают,
		// вопрос ответит 503 с причиной.
		fmt.Fprintln(os.Stderr, "модель не настроена:", err)
		cl, model = nil, ""
	}
	set, err := parseFeatures(*featFlag)
	if err != nil {
		return err
	}
	data := *dataFlag
	if data == "" {
		// Копия каталога курса (сценарий записи) держит базу в <корень>/data.
		if _, err := os.Stat(dbPath(filepath.Join(root.Dir, "data"))); err == nil {
			if _, err := os.Stat(dbPath(dataDir(root, ""))); err != nil {
				data = filepath.Join(root.Dir, "data")
			}
		}
	}
	d, closeDB, err := curatorDeps(ctx, root.Dir, data, *indexFlag)
	if err != nil {
		return err
	}
	defer closeDB()
	d.DocsSearch = docsSearchFor(d)

	s := &web.Server{Deps: d, LLM: cl, Model: model, Features: set}
	s.State = func(ctx context.Context) (web.State, error) {
		st := web.State{Model: model}
		if cl == nil {
			st.Note = "модель не настроена: DEEPSEEK_API_KEY или -llm local"
		}
		if b, err := (gitx.Repo{Dir: root.Dir}).Branch(ctx); err == nil {
			st.Branch = strings.TrimSpace(b)
		}
		if d.Store == nil {
			st.Stale = true
			st.Note = joinNotes(st.Note, "база документации не собрана: curator index")
			return st, nil
		}
		if list, err := d.Store.Docs(ctx); err == nil {
			st.Index.Docs = len(list)
		}
		if idx, err := d.Store.Index(ctx, d.Index); err == nil {
			st.Index.Chunks = idx.Chunks
		}
		if meta, err := docs.ReadMeta(ctx, d.Store); err == nil {
			st.Index.Commit = meta.Commit
			if st.Index.Docs == 0 {
				st.Index.Docs = meta.Docs
			}
		}
		stale, why, err := docs.Stale(ctx, d.Store, root)
		if err != nil {
			st.Note = joinNotes(st.Note, "свежесть индекса: "+err.Error())
		} else {
			st.Stale = stale
			if stale {
				st.Note = joinNotes(st.Note, why)
			}
		}
		return st, nil
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	fmt.Printf("куратор: http://%s/ · корень %s · модель %s · механизмы %s\n", ln.Addr(), root.Dir, orNone(model), set)
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ensureGit — корень без .git становится репозиторием с веткой main и
// одним коммитом «snapshot»: автор задаётся на команде, глобальный config
// не трогается.
func ensureGit(ctx context.Context, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil
	}
	if err := gitx.Available(); err != nil {
		return err
	}
	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if err := run("init", "-q"); err != nil {
		return err
	}
	if err := run("symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
		return err
	}
	if err := run("add", "-A"); err != nil {
		return err
	}
	if err := run("-c", "user.name=curator", "-c", "user.email=curator@local", "commit", "-q", "-m", "snapshot", "--no-verify"); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "git: %s — репозиторий создан, ветка main\n", dir)
	return nil
}

func joinNotes(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + "; " + b
}
