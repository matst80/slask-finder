package persistance

import "tornberg.me/facet-search/pkg/index"

type IndexPersister struct {
	FilePersistance
}

func NewIndexPersister(fp FilePersistance) *IndexPersister {
	return &IndexPersister{fp}
}

func (ip *IndexPersister) LoadIndex(idx *index.Index) error {

	//return ip.Load(idx)
	return nil
}
