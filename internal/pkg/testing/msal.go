package testing

import (
	"fmt"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/goh-chunlin/go-onedrive/onedrive"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	clientId = "b730a7df-d993-4536-bc13-b5d5e5430bbc"
	secret   = "5UY8Q~3bUT8nQmhPokBfksQ0GtodToS4RDzqhaed"
	authCode = "0.AWsAhGcOaINWgkGQjTExuQ_VDt-nMLeT2TZFvBO11eVDC7xrAPs.AgABAAIAAAD--DLA3VO7QrddgJg7WevrAgDs_wQA9P8GqpLnqxEIp3ttQk6W7SUxSeGyVujOTKX69rQ87wN2-XUyyIGWRq5hbx7BfoGV-xE8S-90G5qQnC1oqRr8oLpX5CXtBOcwcL6Ne0z7SglHHYszB_d37nF5VTTo9CMZdv1ht8ano6-mzx4cJpOsWzeLbihBt201H3XPAoxwITMWUjR6cwhVsVUACcQwgiL48wmQYTuB1O04FRlIArvzGh4RqynT6pwZ9eLcUvTVHXLdwa-6Y1tuHyJ6ONMxa8Q79_NVhDkkvBmh2lIX6n6D-ImDwsuFGSUDJ8PYphq7v12Gpf9_l6LNcF1mireVvGtMI5VDiFBvgV2mdb2T0MxNWBH47xQQxORNhUtfqpf62ZMDLYEOZbvWcsLsAy1QlbVnyY4tSgO6r-yk7GKB-izv8bdbz9f-YLuJGVl0HFZWFPdfZMFwCs8oKiYlyXB7gA6eBVWwaGYsPeT8JqdzM3Ghnii6FC_PIu7tfncUXkxYzSYKhj3jaOuizoUNQo2kveS__3KjMj7p47Dfguh2Ad00k0oWsffAKNi_K3X1MPIOHYORXhhsjZJ9My9gFJlKrR6LatH73An7MCnFtYy5y9w_3EsDzcul1lTfl089UdNjFcB33h1a4FBUgdrO-hWg45hr5uwS2UoTkdQ"
)

var accessToken string

func Login() error {
	// Initializing the client credential
	cred, err := confidential.NewCredFromSecret(secret)
	if err != nil {
		return fmt.Errorf("could not create a cred from a secret: %w", err)
	} else {
		fmt.Printf("cred created:%v\r\n", cred)
	}
	confidentialClientApp, err := confidential.New(clientId, cred, confidential.WithAuthority("https://login.microsoftonline.com/common"))
	if err != nil {
		return fmt.Errorf("could not create a app from a credentials: %w", err)
	} else {
		fmt.Printf("client created:%v\r\n", confidentialClientApp)
	}
	authRes, err := confidentialClientApp.AcquireTokenByCredential(context.Background(), []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return fmt.Errorf("could not auth from a credentials: %w", err)
	}
	fmt.Println("Token:" + authRes.AccessToken)
	accessToken = authRes.AccessToken
	fmt.Println(authRes)
	return nil
}

func Login2() error {
	cred, err := confidential.NewCredFromSecret(secret)
	if err != nil {
		return fmt.Errorf("could not create a cred from a secret: %w", err)
	} else {
		fmt.Printf("cred created:%v\r\n", cred)
	}
	confidentialClientApp, err := confidential.New(clientId, cred, confidential.WithAuthority("https://login.microsoftonline.com/common"))
	if err != nil {
		return fmt.Errorf("could not create a app from a credentials: %w", err)
	} else {
		fmt.Printf("client created:%v\r\n", confidentialClientApp)
	}
	auth2PromptUrl, err := confidentialClientApp.AuthCodeURL(context.Background(), clientId, "http://localhost/myapp/", []string{"Files.ReadWrite.All"})
	if err != nil {
		return fmt.Errorf("could not create a redirectUrl: %w", err)
	} else {
		fmt.Printf("auth2 code prompt URL created:\r\n%s\r\n", auth2PromptUrl)
	}
	fmt.Println("Please enter AuthCode:")
	//Scanln doesn't work in testing.
	// fmt.Scanln(&authCode)
	authRes, err := confidentialClientApp.AcquireTokenByAuthCode(context.Background(), authCode, "http://localhost/myapp/", []string{"Files.ReadWrite.All"})
	if err != nil {
		return fmt.Errorf("could not auth from a credentials: %w", err)
	}
	fmt.Println("Token:" + authRes.AccessToken)
	accessToken = authRes.AccessToken
	fmt.Println(authRes)
	return nil
}

func Storage() error {
	ctx := context.Background()
	fmt.Println("Using token: " + accessToken)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := onedrive.NewClient(tc)

	// list all OneDrive drives for the current logged in user
	drives, err := client.Drives.List(ctx)
	if err != nil {
		return err
	}
	fmt.Println(drives.Drives)
	return nil

}
