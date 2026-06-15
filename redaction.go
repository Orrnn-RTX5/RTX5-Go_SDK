package rtx5sdk

import "fmt"

func (r ManagerLoginRequest) String() string {
	return fmt.Sprintf("ManagerLoginRequest{User:%q Password:<redacted> Server:%q BrokerID:%q}", r.User, r.Server, r.BrokerID)
}

func (r ManagerLoginRequest) GoString() string {
	return r.String()
}

func (r CreateAccountRequest) String() string {
	return fmt.Sprintf("CreateAccountRequest{MasterPassword:<redacted> Group:%q InvestorPassword:%s Email:%q}", r.Group, redactPresence(r.InvestorPassword), r.Email)
}

func (r CreateAccountRequest) GoString() string {
	return r.String()
}

func (s Session) String() string {
	return fmt.Sprintf("Session{Token:<redacted> BrokerID:%q Raw:<redacted>}", s.BrokerID)
}

func (s Session) GoString() string {
	return s.String()
}

func redactPresence(value string) string {
	if value == "" {
		return "<empty>"
	}
	return "<redacted>"
}
