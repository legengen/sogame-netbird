//go:build windows

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	releasebuild "github.com/legengen/sogame-netbird/client/build"
)

type WindowsSignatureVerifier struct{}

func (WindowsSignatureVerifier) Verify(ctx context.Context, path string, expected releasebuild.Publisher) error {
	if err := verifyWindowsTrust(path); err != nil {
		return err
	}
	subject, err := authenticodeSubject(ctx, path)
	if err != nil {
		return err
	}
	if !hasDNValue(subject, "CN", expected.SubjectCommonName) || !hasDNValue(subject, "O", expected.Organization) {
		return fmt.Errorf("%w: subject %q", ErrPublisherMismatch, subject)
	}
	return nil
}

func verifyWindowsTrust(path string) error {
	pathUTF16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("encode artifact path: %w", err)
	}
	fileInfo := &windows.WinTrustFileInfo{
		Size:     uint32(unsafe.Sizeof(windows.WinTrustFileInfo{})),
		FilePath: pathUTF16,
	}
	data := &windows.WinTrustData{
		Size:                            uint32(unsafe.Sizeof(windows.WinTrustData{})),
		UIChoice:                        windows.WTD_UI_NONE,
		RevocationChecks:                windows.WTD_REVOKE_NONE,
		UnionChoice:                     windows.WTD_CHOICE_FILE,
		StateAction:                     windows.WTD_STATEACTION_VERIFY,
		ProvFlags:                       windows.WTD_REVOCATION_CHECK_NONE,
		FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(fileInfo),
	}
	verifyErr := windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)
	data.StateAction = windows.WTD_STATEACTION_CLOSE
	_ = windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)
	if verifyErr != nil {
		return fmt.Errorf("%w: %v", ErrSignatureInvalid, verifyErr)
	}
	return nil
}

func authenticodeSubject(ctx context.Context, path string) (string, error) {
	const script = `$ErrorActionPreference='Stop'; $s=Get-AuthenticodeSignature -LiteralPath $env:SOGAME_SIGNATURE_PATH; [Console]::Out.Write(([pscustomobject]@{Status=$s.Status.ToString();Subject=$s.SignerCertificate.Subject}|ConvertTo-Json -Compress))`
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	command.Env = append(os.Environ(), "SOGAME_SIGNATURE_PATH="+path)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read Authenticode publisher: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var result struct {
		Status  string `json:"Status"`
		Subject string `json:"Subject"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("parse Authenticode publisher: %w", err)
	}
	if !strings.EqualFold(result.Status, "Valid") || result.Subject == "" {
		return "", fmt.Errorf("%w: status %s", ErrSignatureInvalid, result.Status)
	}
	return result.Subject, nil
}

func hasDNValue(subject, key, value string) bool {
	pattern := `(?i)(?:^|,\s*)` + regexp.QuoteMeta(key) + `=` + regexp.QuoteMeta(value) + `(?:,|$)`
	return regexp.MustCompile(pattern).MatchString(subject)
}
