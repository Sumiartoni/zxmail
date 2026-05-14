package billing

import "errors"

var ErrUnsupportedPaymentProvider = errors.New("unsupported payment provider")

type PaymentProvider interface {
	Code() string
	DisplayName() string
}

type ManualBankTransferProvider struct{}

func (ManualBankTransferProvider) Code() string {
	return "manual_bank_transfer"
}

func (ManualBankTransferProvider) DisplayName() string {
	return "Manual bank transfer"
}

type ManualQRISProvider struct{}

func (ManualQRISProvider) Code() string {
	return "manual_qris"
}

func (ManualQRISProvider) DisplayName() string {
	return "Manual QRIS"
}

func SupportedProviders() []PaymentProvider {
	return []PaymentProvider{
		ManualBankTransferProvider{},
		ManualQRISProvider{},
	}
}

func IsSupportedProvider(code string) bool {
	for _, provider := range SupportedProviders() {
		if provider.Code() == code {
			return true
		}
	}
	return false
}
