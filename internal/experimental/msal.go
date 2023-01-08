package experimental

import (
	"fmt"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/goh-chunlin/go-onedrive/onedrive"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"os"
)

var accessToken string
var onedriveClient *onedrive.Client
var clientId string
var secret string

func Init() {
	clientId = os.Getenv("CLIENT_ID")
	secret = os.Getenv("SECRET")
}

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
	fmt.Println("DiscordToken:" + authRes.AccessToken)
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
	auth2PromptUrl, err := confidentialClientApp.AuthCodeURL(context.Background(), clientId, "http://localhost/myapp/", []string{"Files.ReadWrite.All", "offline_access"})
	if err != nil {
		return fmt.Errorf("could not create a redirectUrl: %w", err)
	} else {
		fmt.Printf("auth2 code prompt URL created:\r\n%s\r\n", auth2PromptUrl)
	}
	// fmt.Scanln(&authCode)
	return nil

}

func LoginGetToken(authCode string) error {
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
	authRes, err := confidentialClientApp.AcquireTokenByAuthCode(context.Background(), authCode, "http://localhost/myapp/", []string{"Files.ReadWrite.All", "offline_access"})
	if err != nil {
		return fmt.Errorf("could not auth from a credentials: %w", err)
	}
	fmt.Println("DiscordToken:" + authRes.AccessToken)
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
	onedriveClient = client
	fmt.Println(drives.Drives[0].Owner)
	fmt.Println(drives.ODataContext)
	fmt.Println(drives.Drives[0].Id)
	return nil
}

func ListFiles() error {
	items, err := onedriveClient.DriveItems.List(context.Background(), "01MYC6HNVG2BYQ5L3VJJDJYZ2V3GOSHWAM")
	if err != nil {
		return errors.Wrap(err, "error retrieving items through basin id")
	}
	for _, item := range items.DriveItems {
		fmt.Println("item:" + item.Name + "||" + item.Id)
	}
	//actually, parentFolderID
	newFolderItem, err := onedriveClient.DriveItems.CreateNewFolder(context.Background(), "", "01MYC6HNVG2BYQ5L3VJJDJYZ2V3GOSHWAM", "Go Generated Folder")
	if err != nil {
		return errors.Wrap(err, "Failed creating experimental folder")
	}
	fmt.Println("experimental folder name:" + newFolderItem.Name + "||" + newFolderItem.Id)
	onedriveClient.DriveItems.Delete(context.Background(), "", newFolderItem.Id)
	err = onedriveClient.DriveItems.Delete(context.Background(), "", newFolderItem.Id)
	if err != nil {
		return errors.Wrap(err, "failed deleting experimental folder")
	}

	return nil
}

func UploadFile() error {
	LocalFilePath := "I:\\Records\\11\\舒拉·布尔京：摸索黑暗中的群像——为什么俄罗斯人支持这场战争？_凤凰网 (2022_11_12 00_38_41).html"
	onedriveClient.DriveItems.UploadNewFile(context.Background(), "", "01MYC6HNVG2BYQ5L3VJJDJYZ2V3GOSHWAM", LocalFilePath)
	return nil
}
