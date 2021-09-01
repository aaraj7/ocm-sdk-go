/*
Copyright (c) 2019 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains tests for the methods that request tokens.

package authentication

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"                         // nolint
	. "github.com/onsi/gomega"                         // nolint
	. "github.com/onsi/gomega/ghttp"                   // nolint
	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Tokens", func() {
	// Context used by the tests:
	var ctx context.Context

	// Server used during the tests:
	var server *Server

	// Name of the temporary file containing the CA for the server:
	var ca string

	BeforeEach(func() {
		// Create the context:
		ctx = context.Background()

		// Create the servers:
		server, ca = MakeTCPTLSServer()
	})

	AfterEach(func() {
		// Stop the servers:
		server.Close()

		// Remove the temporary CA files:
		err := os.Remove(ca)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Refresh grant", func() {
		It("Returns the access token generated by the server", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
			Expect(returnedRefresh).To(Equal(refreshToken))
		})

		It("Sends the token request the first time only", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens the first time:
			firstAccess, firstRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Get the tones the second time:
			secondAccess, secondRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(firstAccess).To(Equal(secondAccess))
			Expect(firstRefresh).To(Equal(secondRefresh))
		})

		It("Refreshes the access token request if it is expired", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Minute)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(validAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(expiredAccess, refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(validAccess))
		})

		It("Uses opaque refresh token to refresh expired access token", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Minute)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := "my_refresh_token"

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(validAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(expiredAccess, refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(validAccess))
		})

		It("Refreshes the access token if it expires in less than one minute", func() {
			// Generate the tokens:
			firstAccess := MakeTokenString("Bearer", 50*time.Second)
			secondAccess := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(secondAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(firstAccess, refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(secondAccess))
		})

		It("Refreshes the access token if it expires in less than specified expiry period", func() {
			// Ask for a token valid for at least 10 minutes
			expiresIn := 10 * time.Minute

			// Generate the tokens:
			firstAccess := MakeTokenString("Bearer", 9*time.Minute)
			secondAccess := MakeTokenString("Bearer", 20*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(secondAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(firstAccess, refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, _, err := wrapper.Tokens(ctx, expiresIn)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(secondAccess))
		})

		It("Fails if the access token is expired and there is no refresh token", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", -5*time.Second)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(accessToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Succeeds if access token expires soon and there is no refresh token", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", 10*time.Second)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(accessToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Fails if the refresh token is expired", func() {
			// Generate the tokens:
			refreshToken := MakeTokenString("Refresh", -5*time.Second)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
		})

		When("The server doesn't return JSON content type", func() {
			It("Adds complete content to error message if it is short", func() {
				// Generate the refresh token:
				refreshToken := MakeTokenString("Refresh", 10*time.Hour)

				// Configure the server:
				for i := 0; i < 100; i++ { // there are going to be several retries
					server.AppendHandlers(
						RespondWith(
							http.StatusServiceUnavailable,
							`Service unavailable`,
							http.Header{
								"Content-Type": []string{
									"text/plain",
								},
							},
						),
					)
				}

				// Create the wrapper:
				wrapper, err := NewTransportWrapper().
					Logger(logger).
					TokenURL(server.URL()).
					TrustedCA(ca).
					Tokens(refreshToken).
					Build(ctx)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err = wrapper.Close()
					Expect(err).ToNot(HaveOccurred())
				}()

				// Try to get the access token:
				ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				_, _, err = wrapper.Tokens(ctx)
				Expect(err).To(HaveOccurred())
				message := err.Error()
				Expect(message).To(ContainSubstring("text/plain"))
				Expect(message).To(ContainSubstring("Service unavailable"))
			})

			It("Adds summary of content if it is too long", func() {
				// Generate the refresh token:
				refreshToken := MakeTokenString("Refresh", 10*time.Hour)

				// Calculate a long message:
				content := fmt.Sprintf("Ver%s long", strings.Repeat("y", 1000))

				// Configure the server:
				server.AppendHandlers(
					RespondWith(
						http.StatusBadRequest,
						content,
						http.Header{
							"Content-Type": []string{
								"text/plain",
							},
						},
					),
				)

				// Create the wrapper:
				wrapper, err := NewTransportWrapper().
					Logger(logger).
					TokenURL(server.URL()).
					TrustedCA(ca).
					Tokens(refreshToken).
					Build(ctx)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err = wrapper.Close()
					Expect(err).ToNot(HaveOccurred())
				}()

				// Try to get the access token:
				_, _, err = wrapper.Tokens(ctx)
				Expect(err).To(HaveOccurred())
				message := err.Error()
				Expect(message).To(ContainSubstring("text/plain"))
				Expect(message).To(ContainSubstring("Veryyyyyy"))
				Expect(message).To(ContainSubstring("..."))
			})
		})

		It("Honors cookies", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Minute)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					RespondWithCookie("mycookie", "myvalue"),
					RespondWithAccessAndRefreshTokens(expiredAccess, refreshToken),
				),
				CombineHandlers(
					VerifyCookie("mycookie", "myvalue"),
					RespondWithAccessAndRefreshTokens(validAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(expiredAccess, refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Request the tokens the first time. This will return an expired access
			// token and a valid refresh token.
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Request the tokens a second time, therefore forcing a refresh grant which
			// should honor the cookies returned in the first attempt:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Password grant", func() {
		It("Returns the access and refresh tokens generated by the server", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
			Expect(returnedRefresh).To(Equal(refreshToken))
		})

		It("Refreshes access token", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Second)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessAndRefreshTokens(expiredAccess, refreshToken),
				),
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(validAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens the first time:
			firstAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(firstAccess).To(Equal(expiredAccess))

			// Get the tokens the second time:
			secondAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(secondAccess).To(Equal(validAccess))
		})

		It("Requests a new refresh token when it expires", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Second)
			expiredRefresh := MakeTokenString("Refresh", -15*time.Second)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			validRefresh := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessAndRefreshTokens(expiredAccess, expiredRefresh),
				),
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessAndRefreshTokens(validAccess, validRefresh),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens the first time:
			_, firstRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(firstRefresh).To(Equal(expiredRefresh))

			// Get the tokens the second time:
			_, secondRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(secondRefresh).To(Equal(validRefresh))
		})

		It("Requests a new refresh token when expires in less than ten seconds", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Second)
			expiredRefresh := MakeTokenString("Refresh", 5*time.Second)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			validRefresh := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessAndRefreshTokens(expiredAccess, expiredRefresh),
				),
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessAndRefreshTokens(validAccess, validRefresh),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens the first time:
			_, firstRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(firstRefresh).To(Equal(expiredRefresh))

			// Get the tokens the second time:
			_, secondRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(secondRefresh).To(Equal(validRefresh))
		})

		It("Fails with wrong user name", func() {
			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("baduser", "mypassword"),
					RespondWithTokenError("bad_user", "Bad user"),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("baduser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Fails with wrong password", func() {
			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "badpassword"),
					RespondWithTokenError("bad_password", "Bad password"),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "badpassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Honors cookies", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Minute)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					RespondWithCookie("mycookie", "myvalue"),
					RespondWithAccessAndRefreshTokens(expiredAccess, refreshToken),
				),
				CombineHandlers(
					VerifyCookie("mycookie", "myvalue"),
					RespondWithAccessAndRefreshTokens(validAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Request the tokens the first time. This will return an expired access
			// token and a valid refresh token.
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Request the tokens a second time, therefore forcing a refresh grant which
			// should honor the cookies returned in the first attempt:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Works if no refresh token is returned", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", 5*time.Minute)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessToken(accessToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
		})

		It("Accepts lower case token type", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("bearer", 5*time.Minute)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessToken(accessToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
		})

		It("Accepts opaque refresh token", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("bearer", 5*time.Minute)
			refreshToken := "my_refresh_token"

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyPasswordGrant("myuser", "mypassword"),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				User("myuser", "mypassword").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
			Expect(returnedRefresh).To(Equal(refreshToken))
		})
	})

	When("Only the access token is provided", func() {
		It("Returns the access token if it hasn't expired", func() {
			// Generate the token:
			accessToken := MakeTokenString("Bearer", 5*time.Minute)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(accessToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
			Expect(returnedRefresh).To(BeEmpty())
		})

		It("Returns an error if the access token has expired", func() {
			// Generate the token:
			accessToken := MakeTokenString("Bearer", -5*time.Minute)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(accessToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
			Expect(returnedAccess).To(BeEmpty())
			Expect(returnedRefresh).To(BeEmpty())
		})
	})

	Describe("Client credentials grant", func() {
		It("Returns the access token generated by the server", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", 5*time.Minute)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyClientCredentialsGrant("myclient", "mysecret"),
					RespondWithAccessToken(accessToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Client("myclient", "mysecret").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the token:
			returnedAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
		})

		It("Refreshes access token", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Second)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyClientCredentialsGrant("myclient", "mysecret"),
					RespondWithAccessToken(validAccess),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Client("myclient", "mysecret").
				Tokens(expiredAccess).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the token:
			returnedAccess, _, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(validAccess))
		})

		It("Fails with wrong client identifier", func() {
			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyClientCredentialsGrant("badclient", "mysecret"),
					RespondWithTokenError("invalid_grant", "Bad client"),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Client("badclient", "mysecret").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Fails with wrong client secret", func() {
			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyClientCredentialsGrant("myclient", "badsecret"),
					RespondWithTokenError("invalid_grant", "Bad secret"),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Client("myclient", "badsecret").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Honours cookies", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Minute)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					RespondWithCookie("mycookie", "myvalue"),
					RespondWithAccessAndRefreshTokens(expiredAccess, refreshToken),
				),
				CombineHandlers(
					VerifyCookie("mycookie", "myvalue"),
					RespondWithAccessAndRefreshTokens(validAccess, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Client("myclient", "mysecret").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Request the tokens the first time. This will return an expired access
			// token and a valid refresh token.
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Request the tokens a second time, therefore forcing a refresh grant which
			// should honor the cookies returned in the first attempt:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Doesn't fail if the server returns a refresh token", func() {
			// Generate the tokens:
			accessToken := MakeTokenString("Bearer", 5*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server:
			server.AppendHandlers(
				CombineHandlers(
					VerifyClientCredentialsGrant("myclient", "mysecret"),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Client("myclient", "mysecret").
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the token:
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(accessToken))
			Expect(returnedRefresh).To(Equal(refreshToken))
		})

		It("Uses client credentials grant even if it has refresh token", func() {
			// Generate the tokens:
			expiredAccess := MakeTokenString("Bearer", -5*time.Minute)
			validAccess := MakeTokenString("Bearer", 5*time.Minute)
			validRefresh := MakeTokenString("Refresh", 10*time.Hour)

			// Configure the server so that it returns a expired access token and a
			// valid refresh token for the first request, and then a valid access token
			// for the second request. In both cases the client should be using the
			// client credentials grant.
			server.AppendHandlers(
				CombineHandlers(
					VerifyClientCredentialsGrant("myclient", "mysecret"),
					RespondWithAccessAndRefreshTokens(expiredAccess, validRefresh),
				),
				CombineHandlers(
					VerifyClientCredentialsGrant("myclient", "mysecret"),
					RespondWithAccessAndRefreshTokens(validAccess, validRefresh),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Client("myclient", "mysecret").
				Tokens(expiredAccess, validRefresh).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Force the initial token request. This will return an expired access token
			// and a valid refresh token, that way when we get the tokens again the
			// wrapper should send another request, but using the client credentials
			// grant and ignoring the refresh token.
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(expiredAccess))
			Expect(returnedRefresh).To(Equal(validRefresh))

			// Force another request:
			returnedAccess, returnedRefresh, err = wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).To(Equal(validAccess))
			Expect(returnedRefresh).To(Equal(validRefresh))
		})
	})

	Describe("Retry for getting access token", func() {
		It("Return access token after a few retries", func() {
			// Generate tokens:
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)
			accessToken := MakeTokenString("Bearer", 5*time.Minute)

			server.AppendHandlers(
				RespondWithContent(
					http.StatusInternalServerError,
					"text/plain",
					"Internal Server Error",
				),
				RespondWithContent(
					http.StatusBadGateway,
					"text/plain",
					"Bad Gateway",
				),
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			returnedAccess, returnedRefresh, err := wrapper.Tokens(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedAccess).ToNot(BeEmpty())
			Expect(returnedRefresh).ToNot(BeEmpty())
		})

		It("Test no retry when status is not http 5xx", func() {
			// Generate tokens:
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)
			accessToken := MakeTokenString("Bearer", 5*time.Minute)

			server.AppendHandlers(
				RespondWithContent(
					http.StatusInternalServerError,
					"text/plain",
					"Internal Server Error",
				),
				RespondWithJSON(
					http.StatusForbidden,
					"{}",
				),
				CombineHandlers(
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Get the tokens:
			_, _, err = wrapper.Tokens(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Honours context timeout", func() {
			// Generate tokens:
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)
			accessToken := MakeTokenString("Bearer", 5*time.Minute)

			// Configure the server with a handler that introduces an
			// artificial delay:
			server.AppendHandlers(
				CombineHandlers(
					http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
						time.Sleep(10 * time.Millisecond)
					}),
					VerifyRefreshGrant(refreshToken),
					RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
				),
			)

			// Create the wrapper:
			wrapper, err := NewTransportWrapper().
				Logger(logger).
				TokenURL(server.URL()).
				TrustedCA(ca).
				Tokens(refreshToken).
				Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = wrapper.Close()
				Expect(err).ToNot(HaveOccurred())
			}()

			// Request the token with a timeout smaller than the artificial
			// delay introduced by the server:
			ctx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
			defer cancel()
			_, _, err = wrapper.Tokens(ctx)

			// The request should fail with a context deadline exceeded error:
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
		})
	})
})

func VerifyPasswordGrant(user, password string) http.HandlerFunc {
	return CombineHandlers(
		VerifyRequest(http.MethodPost, "/"),
		VerifyContentType("application/x-www-form-urlencoded"),
		VerifyFormKV("grant_type", "password"),
		VerifyFormKV("username", user),
		VerifyFormKV("password", password),
	)
}

func VerifyClientCredentialsGrant(id, secret string) http.HandlerFunc {
	return CombineHandlers(
		VerifyRequest(http.MethodPost, "/"),
		VerifyContentType("application/x-www-form-urlencoded"),
		VerifyFormKV("grant_type", "client_credentials"),
		VerifyFormKV("client_id", id),
		VerifyFormKV("client_secret", secret),
	)
}

func VerifyRefreshGrant(refreshToken string) http.HandlerFunc {
	return CombineHandlers(
		VerifyRequest(http.MethodPost, "/"),
		VerifyContentType("application/x-www-form-urlencoded"),
		VerifyFormKV("grant_type", "refresh_token"),
		VerifyFormKV("refresh_token", refreshToken),
	)
}
