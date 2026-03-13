package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	db "github.com/zkyrnx11/mack/src/sql"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	anticallMu      sync.RWMutex
	anticallEnabled bool
	anticallMode    = "warn"
	anticallCodes   []string
)

var whatsappCallingCodes = map[string]string{
	"1": "USA / Canada", "7": "Russia / Kazakhstan",
	"20": "Egypt", "27": "South Africa",
	"30": "Greece", "31": "Netherlands", "32": "Belgium",
	"33": "France", "34": "Spain", "36": "Hungary",
	"39": "Italy", "40": "Romania", "41": "Switzerland",
	"43": "Austria", "44": "United Kingdom", "45": "Denmark",
	"46": "Sweden", "47": "Norway", "48": "Poland",
	"49": "Germany", "51": "Peru", "52": "Mexico",
	"53": "Cuba", "54": "Argentina", "55": "Brazil",
	"56": "Chile", "57": "Colombia", "58": "Venezuela",
	"60": "Malaysia", "61": "Australia", "62": "Indonesia",
	"63": "Philippines", "64": "New Zealand", "65": "Singapore",
	"66": "Thailand", "81": "Japan", "82": "South Korea",
	"84": "Vietnam", "86": "China", "90": "Turkey",
	"91": "India", "92": "Pakistan", "93": "Afghanistan",
	"94": "Sri Lanka", "95": "Myanmar", "98": "Iran",
	"212": "Morocco", "213": "Algeria", "216": "Tunisia",
	"218": "Libya", "220": "Gambia", "221": "Senegal",
	"222": "Mauritania", "223": "Mali", "224": "Guinea",
	"225": "Ivory Coast", "226": "Burkina Faso", "227": "Niger",
	"228": "Togo", "229": "Benin", "230": "Mauritius",
	"231": "Liberia", "232": "Sierra Leone", "233": "Ghana",
	"234": "Nigeria", "235": "Chad", "236": "Central African Republic",
	"237": "Cameroon", "238": "Cape Verde", "239": "Sao Tome and Principe",
	"240": "Equatorial Guinea", "241": "Gabon", "242": "Republic of the Congo",
	"243": "DR Congo", "244": "Angola", "245": "Guinea-Bissau",
	"248": "Seychelles", "249": "Sudan", "250": "Rwanda",
	"251": "Ethiopia", "252": "Somalia", "253": "Djibouti",
	"254": "Kenya", "255": "Tanzania", "256": "Uganda",
	"257": "Burundi", "258": "Mozambique", "260": "Zambia",
	"261": "Madagascar", "262": "Reunion / Mayotte", "263": "Zimbabwe",
	"264": "Namibia", "265": "Malawi", "266": "Lesotho",
	"267": "Botswana", "268": "Eswatini", "269": "Comoros",
	"291": "Eritrea", "297": "Aruba", "298": "Faroe Islands",
	"299": "Greenland", "350": "Gibraltar", "351": "Portugal",
	"352": "Luxembourg", "353": "Ireland", "354": "Iceland",
	"355": "Albania", "356": "Malta", "357": "Cyprus",
	"358": "Finland", "359": "Bulgaria", "370": "Lithuania",
	"371": "Latvia", "372": "Estonia", "373": "Moldova",
	"374": "Armenia", "375": "Belarus", "376": "Andorra",
	"377": "Monaco", "380": "Ukraine", "381": "Serbia",
	"382": "Montenegro", "383": "Kosovo", "385": "Croatia",
	"386": "Slovenia", "387": "Bosnia and Herzegovina", "389": "North Macedonia",
	"420": "Czech Republic", "421": "Slovakia", "423": "Liechtenstein",
	"500": "Falkland Islands", "501": "Belize", "502": "Guatemala",
	"503": "El Salvador", "504": "Honduras", "505": "Nicaragua",
	"506": "Costa Rica", "507": "Panama", "509": "Haiti",
	"590": "Guadeloupe", "591": "Bolivia", "592": "Guyana",
	"593": "Ecuador", "594": "French Guiana", "595": "Paraguay",
	"596": "Martinique", "597": "Suriname", "598": "Uruguay",
	"599": "Curacao", "670": "East Timor", "673": "Brunei",
	"674": "Nauru", "675": "Papua New Guinea", "676": "Tonga",
	"677": "Solomon Islands", "678": "Vanuatu", "679": "Fiji",
	"680": "Palau", "682": "Cook Islands", "685": "Samoa",
	"686": "Kiribati", "687": "New Caledonia", "688": "Tuvalu",
	"689": "French Polynesia", "691": "Micronesia", "692": "Marshall Islands",
	"850": "North Korea", "852": "Hong Kong", "853": "Macau",
	"855": "Cambodia", "856": "Laos", "880": "Bangladesh",
	"886": "Taiwan", "960": "Maldives", "961": "Lebanon",
	"962": "Jordan", "963": "Syria", "964": "Iraq",
	"965": "Kuwait", "966": "Saudi Arabia", "967": "Yemen",
	"968": "Oman", "970": "Palestine", "971": "UAE",
	"972": "Israel", "973": "Bahrain", "974": "Qatar",
	"975": "Bhutan", "976": "Mongolia", "977": "Nepal",
	"992": "Tajikistan", "993": "Turkmenistan", "994": "Azerbaijan",
	"995": "Georgia", "996": "Kyrgyzstan", "998": "Uzbekistan",
}

type CallOfferHook func(client *whatsmeow.Client, evt *events.CallOffer)

var callOfferHooks []CallOfferHook

func RegisterCallOfferHook(fn CallOfferHook) {
	callOfferHooks = append(callOfferHooks, fn)
}

func init() {
	Register(&Command{
		Pattern:  "anticall",
		IsSudo:   true,
		Category: "utility",
		Func:     anticallCmd,
	})
	RegisterCallOfferHook(anticallHook)
}

func anticallCmd(ctx *Context) error {
	args := ctx.Args

	if len(args) == 0 {
		anticallMu.RLock()
		enabled := anticallEnabled
		mode := anticallMode
		codes := append([]string(nil), anticallCodes...)
		anticallMu.RUnlock()

		status := "off"
		if enabled {
			status = "on"
		}
		codesStr := "all countries"
		if len(codes) > 0 {
			codesStr = strings.Join(codes, ", ")
		}
		ctx.Reply(menuHeader("anticall") + T().AnticallUsage + "\n\n" +
			fmt.Sprintf("Status: *%s*\nMode: *%s*\nCountry codes: %s", status, mode, codesStr))
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "on":
		anticallMu.Lock()
		anticallEnabled = true
		anticallMu.Unlock()
		saveAnticallSettings()
		ctx.Reply(T().AnticallOn)

	case "off":
		anticallMu.Lock()
		anticallEnabled = false
		anticallMu.Unlock()
		saveAnticallSettings()
		ctx.Reply(T().AnticallOff)

	case "set":
		if len(args) < 2 {
			anticallMu.RLock()
			codes := append([]string(nil), anticallCodes...)
			anticallMu.RUnlock()
			msg := menuHeader("anticall set") + T().AnticallSetUsage
			if len(codes) > 0 {
				msg += "\n\n*Blocked codes:* " + strings.Join(codes, ", ")
			} else {
				msg += "\n\nNo codes set - all countries are affected."
			}
			ctx.Reply(msg)
			return nil
		}
		code := strings.TrimPrefix(args[1], "+")
		country, valid := whatsappCallingCodes[code]
		if !valid {
			ctx.Reply(fmt.Sprintf(T().AnticallInvalidCode, args[1]))
			return nil
		}
		anticallMu.Lock()
		removed := false
		for i, c := range anticallCodes {
			if c == code {
				anticallCodes = append(anticallCodes[:i], anticallCodes[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			anticallCodes = append(anticallCodes, code)
		}
		anticallMu.Unlock()
		saveAnticallSettings()
		if removed {
			ctx.Reply(fmt.Sprintf(T().AnticallCodeRemoved, code))
		} else {
			ctx.Reply(fmt.Sprintf(T().AnticallCodeAdded, code, country))
		}

	case "mode":
		if len(args) < 2 {
			anticallMu.RLock()
			mode := anticallMode
			anticallMu.RUnlock()
			ctx.Reply(menuHeader("anticall mode") +
				T().AnticallModeUsage + "\n\nCurrent mode: *" + mode + "*")
			return nil
		}
		newMode := strings.ToLower(args[1])
		if newMode != "block" && newMode != "warn" {
			ctx.Reply(menuHeader("anticall mode") + T().AnticallModeUsage)
			return nil
		}
		anticallMu.Lock()
		anticallMode = newMode
		anticallMu.Unlock()
		saveAnticallSettings()
		ctx.Reply(fmt.Sprintf(T().AnticallModeSet, newMode))

	default:
		ctx.Reply(menuHeader("anticall") + T().AnticallUsage)
	}
	return nil
}

func anticallHook(client *whatsmeow.Client, evt *events.CallOffer) {
	anticallMu.RLock()
	enabled := anticallEnabled
	mode := anticallMode
	codes := append([]string(nil), anticallCodes...)
	anticallMu.RUnlock()

	if !enabled {
		return
	}

	callerPhone := evt.From.User
	callerJID := evt.From.ToNonAD()

	if evt.From.Server == types.HiddenUserServer {
		if phone := GetAltID(evt.From.User + "@" + evt.From.Server); phone != "" {
			callerPhone = phone
		}
	}
	callerJIDStr := callerJID.String()

	if len(codes) > 0 {
		code := matchCallingCode(callerPhone)
		matched := false
		for _, c := range codes {
			if c == code {
				matched = true
				break
			}
		}
		if !matched {
			return
		}
	}

	client.RejectCall(context.Background(), evt.From, evt.CallID)

	mentionText := func(tmpl string) string {
		return fmt.Sprintf(tmpl, "@"+callerPhone)
	}
	dmToUser := func(text string) {
		sendMentionToChat(client, callerJID, text, []string{callerJIDStr})
	}

	switch mode {
	case "warn":
		dmToUser(mentionText(T().AnticallCallerDisabled))

	case "block":
		if !db.IsAnticallWarned(callerPhone) {
			db.SetAnticallWarned(callerPhone, true)
			dmToUser(mentionText(T().AnticallCallerBlockWarn))
		} else {
			db.SetAnticallWarned(callerPhone, false)
			dmToUser(mentionText(T().AnticallCallerBlocked))
		}
	}
}

func matchCallingCode(phone string) string {
	for _, length := range []int{3, 2, 1} {
		if len(phone) >= length {
			prefix := phone[:length]
			if _, ok := whatsappCallingCodes[prefix]; ok {
				return prefix
			}
		}
	}
	return ""
}

func loadAnticallSettings() {
	rows := db.GetAnticallRows()
	anticallMu.Lock()
	defer anticallMu.Unlock()
	for k, v := range rows {
		switch k {
		case "enabled":
			anticallEnabled = v == "true"
		case "mode":
			if v == "block" || v == "warn" {
				anticallMode = v
			}
		case "codes":
			var c []string
			if json.Unmarshal([]byte(v), &c) == nil {
				anticallCodes = c
			}
		}
	}
}

func saveAnticallSettings() {
	anticallMu.RLock()
	enabled := anticallEnabled
	mode := anticallMode
	codes := append([]string(nil), anticallCodes...)
	anticallMu.RUnlock()

	enabledStr := "false"
	if enabled {
		enabledStr = "true"
	}
	codesJSON, _ := json.Marshal(codes)

	db.SaveAnticallRows([][2]string{
		{"enabled", enabledStr},
		{"mode", mode},
		{"codes", string(codesJSON)},
	})
}
