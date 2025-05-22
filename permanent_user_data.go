package y_crdt

type PermanentUserData struct {
	YUsers  IAbstractType
	Doc     *Doc
	Clients map[Number]string
	Dss     map[string]*DeleteSet
}

func (p *PermanentUserData) SetUserMapping(doc *Doc, clientID Number, userDescription string, filer func(trans *Transaction, set *DeleteSet) bool) {
	users := p.YUsers.(*YMap)
	user, ok := users.Get(userDescription).(*YMap)
	if !ok {
		user = NewYMap(nil)
		user.Set("ids", NewYArray())
		user.Set("ds", NewYArray())
		users.Set(userDescription, user)
	}

	a := user.Get("ids").(*YArray)
	a.Push(ArrayAny{clientID})
	users.Observe(func(e interface{}, t interface{}) {
		userOverWrite := users.Get(userDescription).(*YMap)
		if userOverWrite != user {
			// user was overwritten, port all data over to the next user object
			// @todo Experiment with Y.Sets here
			user = userOverWrite

			// @todo iterate over old type
			for clientID, _userDescription := range p.Clients {
				if userDescription == _userDescription {
					a := user.Get("ids").(*YArray)
					a.Push(ArrayAny{clientID})
				}
			}

			encoder := NewUpdateEncoderV1()
			ds := p.Dss[userDescription]
			if ds != nil {
				WriteDeleteSet(encoder, ds)
				a := user.Get("ds").(*YArray)
				a.Push(ArrayAny{encoder.ToUint8Array()})
			}
		}
	})

	doc.On("afterTransaction", NewObserverHandler(func(v ...interface{}) {
		trans := v[0].(*Transaction)
		yds := user.Get("ds").(*YArray)
		ds := trans.DeleteSet
		if trans.Local && len(ds.Clients) > 0 && filer(trans, ds) {
			encoder := NewUpdateEncoderV1()
			WriteDeleteSet(encoder, ds)
			yds.Push(ArrayAny{encoder.ToUint8Array()})
		}
	}))
}

func (p *PermanentUserData) GetUserByClientID(clientID Number) string {
	return p.Clients[clientID]
}

func (p *PermanentUserData) GetUserByDeletedID(id *ID) string {
	for userDescription, ds := range p.Dss {
		if IsDeleted(ds, id) {
			return userDescription
		}
	}

	return ""
}

func NewPermanentUserData(doc *Doc, storeType IAbstractType) *PermanentUserData {
	if storeType == nil {
		storeType = doc.GetMap("users")
	}

	dss := make(map[string]*DeleteSet)

	p := &PermanentUserData{
		YUsers:  storeType,
		Doc:     doc,
		Clients: make(map[Number]string),
		Dss:     dss,
	}

	initUser := func(user *YMap, userDescription string) {
		ds, _ := user.Get("ds").(*YArray)
		ids, _ := user.Get("ids").(*YArray)

		if ds == nil || ids == nil {
			return
		}

		addClientId := func(clientID Number) {
			p.Clients[clientID] = userDescription
		}

		ds.ObserveDeep(func(e interface{}, t interface{}) {
			event := e.(*YArrayEvent)
			a := event.Changes["added"].(Set)
			a.Range(func(element interface{}) {
				item, ok := element.(*Item)
				if ok {
					for _, encodeDs := range item.Content.GetContent() {
						data, ok := encodeDs.([]uint8)
						if ok {
							delSet, exist := p.Dss[userDescription]
							if !exist {
								delSet = NewDeleteSet()
							}
							p.Dss[userDescription] = MergeDeleteSets([]*DeleteSet{delSet, ReadDeleteSet(NewUpdateDecoderV1(data))})
						}
					}
				}
			})
		})

		var delSet []*DeleteSet
		ds.Map(func(encodeDs interface{}, number Number, abstractType IAbstractType) interface{} {
			data, ok := encodeDs.([]uint8)
			if ok {
				delSet = append(delSet, ReadDeleteSet(NewUpdateDecoderV1(data)))
			}
			return nil
		})

		p.Dss[userDescription] = MergeDeleteSets(delSet)
		ids.Observe(func(e interface{}, t interface{}) {
			event, ok := e.(*YArrayEvent)
			if !ok {
				return
			}
			a := event.GetChanges()["added"].(Set)
			a.Range(func(element interface{}) {
				item, ok := element.(*Item)
				if ok {
					arr := item.Content.GetContent()
					for _, j := range arr {
						n, _ := j.(Number)
						addClientId(n)
					}
				}
			})
		})

		ids.ForEach(func(i interface{}, number Number, abstractType IAbstractType) {
			clientID := i.(Number)
			addClientId(clientID)
		})
	}

	storeType.Observe(func(e interface{}, t interface{}) {
		event := e.(*YMapEvent)
		a := event.KeysChanged
		a.Range(func(element interface{}) {
			userDescription := element.(string)
			if m, ok := storeType.(*YMap); ok {
				if user, ok := m.Get(userDescription).(*YMap); ok {
					initUser(user, userDescription)
				} else {
					Logf("cannot get user. storeType:%+v userDescription:%s", storeType, userDescription)
				}
			} else {
				Logf("storeType is not *YMap. storeType:%+v", storeType)
			}
		})
	})

	storeType.(*YMap).ForEach(func(s string, i interface{}, yMap *YMap) {
		initUser(yMap, s)
	})

	return p
}
