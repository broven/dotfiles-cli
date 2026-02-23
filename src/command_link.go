package dotfiles

func Link(repoInput string, specified []string, dry bool) error {
	repo, err := absolutePathToRepo(repoInput)
	if err != nil {
		return err
	}

	cfg, err := GetConfig(repo)
	if err != nil {
		return err
	}

	m := cfg.Mappings
	var linkErr error
	if len(specified) == 0 {
		linkErr = m.CreateAllLinks(repo, dry)
		if e, ok := linkErr.(*NothingLinkedError); ok {
			e.RepoPath = repo.String()
		}
	} else {
		linkErr = m.CreateSomeLinks(specified, repo, dry)
	}

	managed, err := syncPackageManagers(cfg.PackageManagers, dry)
	if err != nil {
		return err
	}

	if _, ok := linkErr.(*NothingLinkedError); ok && managed {
		return nil
	}

	return linkErr
}
