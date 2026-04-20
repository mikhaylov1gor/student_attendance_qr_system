package user

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// tempPasswordAlphabet — безопасный алфавит для генерируемых temp-паролей.
// Сознательно исключены: 0, O (ноль/о), 1, l, I (единица/л/И), которые
// путаются при надиктовке или вводе. Знаки препинания тоже убраны — проще
// диктовать и вставлять в мобильную клавиатуру.
const tempPasswordAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"

// TempPasswordLen — длина temp-пароля. 12 символов из 54-символьного алфавита
// дают ≈ 68 бит энтропии — заведомо достаточно для одноразового пароля,
// который пользователь сменит при первом входе (стадия 13+).
const TempPasswordLen = 12

// GenerateTempPassword возвращает криптографически случайную строку из
// tempPasswordAlphabet длиной TempPasswordLen. Используется при Create и
// ResetPassword в user-сервисе; значение показывается админу один раз.
func GenerateTempPassword() (string, error) {
	return generateFrom(tempPasswordAlphabet, TempPasswordLen)
}

func generateFrom(alphabet string, n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("temppass: length must be > 0")
	}
	max := big.NewInt(int64(len(alphabet)))
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("temppass: rand: %w", err)
		}
		out[i] = alphabet[idx.Int64()]
	}
	return string(out), nil
}
