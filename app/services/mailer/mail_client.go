// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mailer

import (
	"crypto/tls"

	gomail "gopkg.in/mail.v2"
)

type Service struct {
	dialer   *gomail.Dialer
	fromMail string
}

func NewMailClient(
	host string,
	port int,
	username string,
	fromMail string,
	password string,
	insecureSkipVerify bool,
) *Service {
	d := gomail.NewDialer(host, port, username, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: insecureSkipVerify} // #nosec G402 (insecure TLS configuration)

	return &Service{
		dialer:   d,
		fromMail: fromMail,
	}
}

func (c *Service) SendMail(mailRequest *MailRequest) error {
	mail := mailRequest.ToGoMail()
	mail.SetHeader("From", c.fromMail)
	return c.dialer.DialAndSend(mail)
}
