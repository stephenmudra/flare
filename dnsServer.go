package main

import (
	"github.com/joyrexus/buckets"
	"github.com/miekg/dns"
	"log"
	"net"
"strings"
)


type DnsServer struct {
	port string
	db *buckets.Bucket
}

func (d DnsServer) tcpServe() {
	dns.HandleFunc(".", d.route)

	server := &dns.Server{Addr: ":" + d.port, Net: "tcp", TsigSecret: nil}
	log.Printf("DNS TCP Serving on port %s", d.port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to setup the %s server: %v\n", ":" + d.port, err)
	}
}

func (d DnsServer) udpServe() {
	dns.HandleFunc(".", d.route)

	server := &dns.Server{Addr: ":" + d.port, Net: "udp", TsigSecret: nil}
	log.Printf("DNS UDP Serving on port %s", d.port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to setup the %s server: %v\n", ":" + d.port, err)
	}
}

func (d DnsServer) route(res dns.ResponseWriter, req *dns.Msg) {
	numQuestions := len(req.Question)

	transport := "udp"
	if _, ok := res.RemoteAddr().(*net.TCPAddr); ok {
		transport = "tcp"
	}

	if numQuestions == 0 {
		log.Printf("FAIL: NO DNS QUESTION\n")
		dns.HandleFailed(res, req)
		return
	}

	if numQuestions == 1 {
		res.WriteMsg(d.handleSingleQuestion(req, transport))
		return
	}
}

func (d DnsServer) getRouteConfig(url string) *RouteConfig {
	for {
		if url == "" {
			url = "."
		}

		value, err := d.db.Get([]byte(url))
		if err == nil && value != nil {
			route := &RouteConfig{}
			err = route.Unmarshal(value)
			if err == nil && route.Active {
				return route
			}
		}

		if url == "." {
			return &RouteConfig{
				Type: "forwarding",
				Nameservers: []string{"8.8.8.8", "8.8.4.4"},
			}
		}

		result := strings.SplitN(url, ".", 2)
		url = result[1]
	}
}

func (d DnsServer) handleSingleQuestion(req *dns.Msg, transport string) *dns.Msg {
	log.Printf("DNS QUESTION: %s \n", req.Question[0].String())
	config := d.getRouteConfig(req.Question[0].Name)
	log.Printf("DNS CONFIG: %s \n", config)

	var msg *dns.Msg
	if config.Type == "forwarding" {
		msg = d.makeNSRequest(config.Nameservers, req, transport)
	} else if config.Type == "static" {
		msg = d.makeStaticRequest(config, req, transport)
	} else {
		msg = d.generateDNSError(req, true)
	}

	return msg
}


func (d DnsServer) generateDNSError(req *dns.Msg, local bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(req)
	m.SetRcode(req, dns.RcodeServerFailure)
	if local {
		m.Authoritative = false     // no matter what set to false
		m.RecursionAvailable = true // and this is still true
	}
	return m
}

func (d DnsServer) makeNSRequest(nameservers []string, req *dns.Msg, transport string) *dns.Msg {
	dnsClient := &dns.Client{Net: transport, SingleInflight: true}
	for _, nameserver := range nameservers {
		if i := strings.Index(nameserver, ":"); i < 0 {
			nameserver += ":53"
		}

		r, _, err := dnsClient.Exchange(req, nameserver)
		if err == nil {
			r.Compress = true
			return r
		}
	}

	log.Printf("Failure to Forward Request\n")
	return d.generateDNSError(req, len(nameservers) == 0)
}

func (d DnsServer) makeStaticRequest(config *RouteConfig, req *dns.Msg, transport string) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(req)

	for _, q := range req.Question {
		for _, Addr := range config.Addrs {
			addr := net.ParseIP(Addr)
			if addr.To4() != nil { // "If ip is not an IPv4 address, To4 returns nil."
				d.dnsAppend(q, m, &dns.A{A: addr})
			} else {
				d.dnsAppend(q, m, &dns.AAAA{AAAA: addr})
			}
		}

		for _, Cname := range config.Cnames {
			cname := dns.Fqdn(Cname)
			d.dnsAppend(q, m, &dns.CNAME{Target: cname})

			if req.RecursionDesired {
				recR := &dns.Msg{
					MsgHdr: dns.MsgHdr{
						Id: dns.Id(),
					},
					Question: []dns.Question{
						{Name: cname, Qtype: q.Qtype, Qclass: q.Qclass},
					},
				}
				recM := d.handleSingleQuestion(recR, transport)
				for _, rr := range recM.Answer {
					d.dnsAppend(q, m, rr)
				}
				for _, rr := range recM.Extra {
					d.dnsAppend(q, m, rr)
				}
			}
		}

		for _, txt := range config.Txts {
			d.dnsAppend(q, m, &dns.TXT{Txt: txt})
		}
	}

	return m
}

func (d DnsServer) dnsAppend(q dns.Question, m *dns.Msg, rr dns.RR) {
	hdr := dns.RR_Header{Name: q.Name, Class: q.Qclass, Ttl: 0}

	if rrS, ok := rr.(*dns.A); ok {
		hdr.Rrtype = dns.TypeA
		rrS.Hdr = hdr
	} else if rrS, ok := rr.(*dns.AAAA); ok {
		hdr.Rrtype = dns.TypeAAAA
		rrS.Hdr = hdr
	} else if rrS, ok := rr.(*dns.CNAME); ok {
		hdr.Rrtype = dns.TypeCNAME
		rrS.Hdr = hdr
	} else if rrS, ok := rr.(*dns.TXT); ok {
		hdr.Rrtype = dns.TypeTXT
		rrS.Hdr = hdr
	} else {
		log.Printf("error: unknown dnsAppend RR type: %+v\n", rr)
		return
	}

	if q.Qtype == dns.TypeANY || q.Qtype == rr.Header().Rrtype {
		m.Answer = append(m.Answer, rr)
	} else {
		m.Extra = append(m.Extra, rr)
	}
}