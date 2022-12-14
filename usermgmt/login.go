package usermgmt

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
)

// https://husobee.github.io/golang/ip-address/2015/12/17/remote-ip-go.html

type ipRange struct {
	start net.IP
	end   net.IP
}

type LoginResponse struct {
	Success     bool `json:"success"`
	Local       bool `json:"local"`
	ChancesLeft int  `json:"chancesLeft"`
}

// inRange - check to see if a given ip address is within a range given
func inRange(r ipRange, ipAddress net.IP) bool {
	// strcmp type byte comparison
	if bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) < 0 {
		return true
	}
	return false
}

var privateRanges = []ipRange{
	ipRange{
		start: net.ParseIP("10.0.0.0"),
		end:   net.ParseIP("10.255.255.255"),
	},
	ipRange{
		start: net.ParseIP("100.64.0.0"),
		end:   net.ParseIP("100.127.255.255"),
	},
	ipRange{
		start: net.ParseIP("172.16.0.0"),
		end:   net.ParseIP("172.31.255.255"),
	},
	ipRange{
		start: net.ParseIP("192.0.0.0"),
		end:   net.ParseIP("192.0.0.255"),
	},
	ipRange{
		start: net.ParseIP("192.168.0.0"),
		end:   net.ParseIP("192.168.255.255"),
	},
	ipRange{
		start: net.ParseIP("198.18.0.0"),
		end:   net.ParseIP("198.19.255.255"),
	},
}

func isPrivateSubnet(ipAddress net.IP) bool {
	// my use case is only concerned with ipv4 atm
	if ipCheck := ipAddress.To4(); ipCheck != nil {
		// iterate over all our ranges
		for _, r := range privateRanges {
			// check if this ip is in a private range
			if inRange(r, ipAddress) {
				return true
			}
		}
	}
	return false
}

func getIPAdress(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")
		// march from right to left until we get a public address
		// that will be the address right before our proxy.
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])
			// header can contain spaces too, strip those out.
			realIP := net.ParseIP(ip)
			if !realIP.IsGlobalUnicast() || isPrivateSubnet(realIP) {
				// bad address, go to next
				continue
			}
			return ip
		}
	}
	return ""
}

func LoginFailedResponse(r *http.Request) LoginResponse {
	oldLimitMS := time.Now().Add(WrongLogInWindow)
	database.DB.Exec("DELETE FROM wrong_login WHERE time < $1", oldLimitMS)

	ip := getIPAdress(r)
	if ip == "" {
		return LoginResponse{
			Success:     false,
			Local:       true,
			ChancesLeft: MaxWrongLogIn,
		}
	}
	var count int
	var response LoginResponse
	if err := database.DB.QueryRow("SELECT COUNT FROM wrong_login WHERE ip = $1", ip).Scan(&count); err != nil {
		response = LoginResponse{
			Success:     false,
			Local:       false,
			ChancesLeft: MaxWrongLogIn,
		}
	} else {
		response = LoginResponse{
			Success:     false,
			Local:       false,
			ChancesLeft: MaxWrongLogIn - count,
		}
	}

	database.DB.Exec("INSERT INTO wrong_login (ip) VALUES ($1)", ip)

	return response
}
