package bars

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

type Client struct {
	httpClient *http.Client
}

// NewClient создаёт нового клиента для взаимодействия с БАРС.
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		httpClient: &http.Client{Jar: jar},
	}
}

// Authorization выполняет авторизацию пользователя по его username и password в системе БАРС.
// Возвращает ErrNoAuth, если авторизация выполнена неуспешно.
func (c *Client) Authorization(ctx context.Context, username, password string) error {
	verificationToken, err := c.getVerificationToken(ctx)
	if err != nil {
		return fmt.Errorf("client authorization: %v", err)
	}

	data := url.Values{
		"__RequestVerificationToken": {verificationToken},
		"UserName":                   {username},
		"Password":                   {password},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, RegistrationURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(response.Request.URL.String(), PersonalGradesPageURL) {
		return ErrWrongGradesPage
	}

	_, err = io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	authstatus := authStatus(response)

	if !authstatus {
		return ErrNoAuth
	}

	return nil
}

// getPage выполняет запрос с установленными заголовками request.Header.
// Возвращает http.Response, если запрос выполнен успешно.
func (c *Client) getPage(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// getVerificationToken возвращает значение verification token cookie.
// Возвращает ошибку, если такой cookie не был предоставлен.
func (c *Client) getVerificationToken(ctx context.Context) (string, error) {
	response, err := c.getPage(ctx, http.MethodPost, RegistrationURL, nil)
	if err != nil {
		return "", fmt.Errorf("registration page was not received: %v", err)
	}

	cookies := response.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == verificationTokenCookie {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("verification token was not provided by BARS")
}

// authStatus проверяет статус авторизации пользователя.
// Возвращает true, если пользователь авторизован, иначе false.
func authStatus(response *http.Response) bool {
	cookies := response.Request.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == authBarsCookie || cookie.Name == sessionCookie {
			return true
		}
	}

	return false
}
