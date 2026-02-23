package dotfiles

func Clean(repoInput string) error {
	repo, err := absolutePathToRepo(repoInput)
	if err != nil {
		return err
	}

	m, err := GetMappings(repo)
	if err != nil {
		return err
	}

	return m.UnlinkAll(repo)
}
