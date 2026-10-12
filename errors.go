package walletsdk

import "errors"

// SDK 哨兵错误，配合 errors.Is 使用。
var (
	// ErrUnsupportedChain 表示链不受支持。
	ErrUnsupportedChain = errors.New("walletsdk: 不支持的链")
	// ErrInvalidMnemonic 表示助记词非法。
	ErrInvalidMnemonic = errors.New("walletsdk: 非法助记词")
	// ErrInvalidPrivateKey 表示私钥非法。
	ErrInvalidPrivateKey = errors.New("walletsdk: 非法私钥")
	// ErrInvalidAddress 表示地址非法。
	ErrInvalidAddress = errors.New("walletsdk: 非法地址")
	// ErrInvalidAmount 表示金额非法。
	ErrInvalidAmount = errors.New("walletsdk: 非法金额")
	// ErrInvalidWordCount 表示助记词词数不合法。
	ErrInvalidWordCount = errors.New("walletsdk: 助记词词数不合法")
	// ErrAssetChainMismatch 表示资产与链不匹配。
	ErrAssetChainMismatch = errors.New("walletsdk: 资产与链不匹配")
	// ErrInsufficientBalance 表示余额不足。
	ErrInsufficientBalance = errors.New("walletsdk: 余额不足")
)
