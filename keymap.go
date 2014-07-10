package peco

import (
	"fmt"
	"os"

	"github.com/nsf/termbox-go"
	"github.com/peco/peco/keyseq"
)

type Keymap struct {
	Config map[string]string
	Action map[string][]string // custom actions
	Keyseq *keyseq.Keyseq
}

func NewKeymap(config map[string]string, actions map[string][]string) Keymap {
	return Keymap{config, actions, keyseq.New()}

}

func (km Keymap) Handler(ev termbox.Event) Action {
	modifier := keyseq.ModNone
	if (ev.Mod & termbox.ModAlt) != 0 {
		modifier = keyseq.ModAlt
	}

	key := keyseq.Key{modifier, ev.Key, ev.Ch}
	action, err := km.Keyseq.AcceptKey(key)

	switch err {
	case nil:
		// Found an action!
		return action.(Action)
	case keyseq.ErrInSequence:
		// TODO We're in some sort of key sequence. Remember what we have
		// received so far
		return ActionFunc(doNothing)
	default:
		return ActionFunc(doAcceptChar)
	}
}

const maxResolveActionDepth = 100
func (km Keymap) resolveActionName(name string, depth int) (Action, error) {
	if depth >= maxResolveActionDepth {
		return nil, fmt.Errorf("Could not resolve %s: deep recursion", name)
	}

	// Can it be resolved via regular nameToActions ?
	v, ok := nameToActions[name]
	if ok {
		return v, nil
	}

	// Can it be resolved via combined actions?
	l, ok := km.Action[name]
	if ok {
		actions := []Action{}
		for _, actionName := range l {
			child, err := km.resolveActionName(actionName, depth + 1)
			if err != nil {
				return nil, err
			}
			actions = append(actions, child)
		}
		v = makeCombinedAction(actions...)
		nameToActions[name] = v
		return v, nil
	}

	return nil, fmt.Errorf("Could not resolve %s: no such action", name)
}

func (km Keymap) ApplyKeybinding() {
	k := km.Keyseq
	k.Clear()

	// Copy the map
	kb := map[string]Action{}
	for s, a := range defaultKeyBinding {
		kb[s] = a
	}

	// munge the map using config
	for s, as := range km.Config {
		if as == "-" {
			delete(kb, s)
			continue
		}

		v, err := km.resolveActionName(as, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		kb[s] = v
	}

	// now compile using kb
	for s, a := range kb {
		list, err := keyseq.ToKeyList(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unknown key %s: %s", s, err)
			continue
		}

		k.Add(list, a)
	}

	k.Compile()
}

// TODO: this needs to be fixed.
func (km Keymap) hasModifierMaps() bool {
	return false
}
