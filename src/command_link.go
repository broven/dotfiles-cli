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
	relink := cfg.Relink
	reqTarget := cfg.RequireTarget
	var linkErr error
	if len(specified) == 0 {
		linkErr = m.CreateAllLinksWithRelink(repo, dry, relink, reqTarget)
		if e, ok := linkErr.(*NothingLinkedError); ok {
			e.RepoPath = repo.String()
		}
	} else {
		linkErr = m.CreateSomeLinksWithRelink(specified, repo, dry, relink, reqTarget)
	}

	return linkErr
}
