// +build !linux

package bindmnt

func resolve(p string) (string, error) {
	return p, nil
}
