// Javascript is generated from typescript, do not modify generated javascript because changes will be overwritten.

// From HTML.
declare let page: HTMLElement
declare let beaconversion: string

const login = async (reason: string) => {
	return new Promise<string>((resolve: (v: string) => void, _) => {
		const origFocus = document.activeElement
		let reasonElem: HTMLElement
		let fieldset: HTMLFieldSetElement
		let password: HTMLInputElement
		const root = dom.div(
			style({position: 'absolute', top: 0, right: 0, bottom: 0, left: 0, backgroundColor: '#eee', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: '1', animation: 'fadein .15s ease-in'}),
			dom.div(
				reasonElem=reason ? dom.div(style({marginBottom: '2ex', textAlign: 'center'}), reason) : dom.div(),
				dom.div(
					style({backgroundColor: 'white', borderRadius: '.25em', padding: '1em', boxShadow: '0 0 20px rgba(0, 0, 0, 0.1)', border: '1px solid #ddd', maxWidth: '95vw', overflowX: 'auto', maxHeight: '95vh', overflowY: 'auto', marginBottom: '20vh'}),
					dom.form(
						async function submit(e: SubmitEvent) {
							e.preventDefault()
							e.stopPropagation()

							reasonElem.remove()

							try {
								fieldset.disabled = true
								const loginToken = await client.LoginPrep()
								const token = await client.Login(loginToken, password.value)
								try {
									window.localStorage.setItem('webadmincsrftoken', token)
								} catch (err) {
									console.log('saving csrf token in localStorage', err)
								}
								root.remove()
								if (origFocus && origFocus instanceof HTMLElement && origFocus.parentNode) {
									origFocus.focus()
								}
								resolve(token)
							} catch (err) {
								console.log('login error', err)
								window.alert('Error: ' + errmsg(err))
							} finally {
								fieldset.disabled = false
							}
						},
						fieldset=dom.fieldset(
							dom.h1('Admin'),
							dom.label(
								style({display: 'block', marginBottom: '2ex'}),
								dom.div('Password', style({marginBottom: '.5ex'})),
								password=dom.input(attr.type('password'), attr.required('')),
							),
							dom.div(
								style({textAlign: 'center'}),
								dom.submitbutton('Login'),
							),
						),
					)
				)
			)
		)
		document.body.appendChild(root)
		password.focus()
	})
}

const localStorageGet = (k: string): string | null => {
	try {
		return window.localStorage.getItem(k)
	} catch (err) {
		return null
	}
}

const localStorageRemove = (k: string) => {
	try {
		return window.localStorage.removeItem(k)
	} catch (err) {
	}
}

const client = new api.Client().withOptions({csrfHeader: 'x-mox-csrf', login: login}).withAuthToken(localStorageGet('webadmincsrftoken') || '')

const green = '#1dea20'
const yellow = '#ffe400'
const red = '#ff7443'
const blue = '#8bc8ff'

const link = (href: string, anchorOpt?: string) => dom.a(attr.href(href), attr.rel('noopener noreferrer'), anchorOpt || href)

const crumblink = (text: string, link: string) => dom.a(text, attr.href(link))
const crumbs = (...l: ElemArg[]) => [
	dom.div(
		style({float: 'right'}),
		dom.clickbutton('Logout', attr.title('Logout, invalidating this session.'), async function click(e: MouseEvent) {
			const b = e.target! as HTMLButtonElement
			try {
				b.disabled = true
				await client.Logout()
			} catch (err) {
				console.log('logout', err)
				window.alert('Error: ' + errmsg(err))
			} finally {
				b.disabled = false
			}

			localStorageRemove('webadmincsrftoken')
			// Reload so all state is cleared from memory.
			window.location.reload()
		}),
	),
	dom.h1(l.map((e, index) => index === 0 ? e : [' / ', e])),
	dom.br()
]

const errmsg = (err: unknown) => ''+((err as any).message || '(no error message)')

const footer = dom.div(
	style({marginTop: '6ex', opacity: 0.75}),
	link('https://github.com/mjl-/mox', 'mox'),
	' ',
	moxversion,
)

const age = (date: Date, future: boolean, nowSecs: number) => {
	if (!nowSecs) {
		nowSecs = new Date().getTime()/1000
	}
	let t = nowSecs - date.getTime()/1000
	let negative = false
	if (t < 0) {
		negative = true
		t = -t
	}
	const minute = 60
	const hour = 60*minute
	const day = 24*hour
	const month = 30*day
	const year = 365*day
	const periods = [year, month, day, hour, minute, 1]
	const suffix = ['y', 'm', 'd', 'h', 'mins', 's']
	let l: string[] = []
	for (let i = 0; i < periods.length; i++) {
		const p = periods[i]
		if (t >= 2*p || i == periods.length-1) {
			const n = Math.floor(t/p)
			l.push('' + n + suffix[i])
			t -= n*p
			if (l.length >= 2) {
				break
			}
		}
	}
	let s = l.join(' ')
	if (!future || !negative) {
		s += ' ago'
	}
	return dom.span(attr.title(date.toString()), s)
}

const domainName = (d: api.Domain) => {
	return d.Unicode || d.ASCII
}

const domainString = (d: api.Domain) => {
	if (d.Unicode) {
		return d.Unicode+" ("+d.ASCII+")"
	}
	return d.ASCII
}

// IP is encoded as base64 bytes, either 4 or 16.
// It's a bit silly to encode this in JS, but it's convenient to simply pass on the
// net.IP bytes from the backend.
const formatIP = (s: string) => {
	const buf = window.atob(s)
	const bytes = Uint8Array.from(buf, (m) => m.codePointAt(0) || 0)
	if (bytes.length === 4 || isIPv4MappedIPv6(bytes)) {
		// Format last 4 bytes as IPv4 address..
		return [bytes.at(-4), bytes.at(-3), bytes.at(-2), bytes.at(-1)].join('.')
	}
	return formatIPv6(bytes)
}

// See if b is of the form ::ffff:x.x.x.x
const v4v6Prefix = new Uint8Array([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff])
const isIPv4MappedIPv6 = (b: Uint8Array) => {
	if (b.length !== 16) {
		return false
	}
	for (let i = 0; i < v4v6Prefix.length; i++) {
		if (b[i] !== v4v6Prefix[i]) {
			return false
		}
	}
	return true
}

const formatIPv6 = (b: Uint8Array) => {
	const hexchars = "0123456789abcdef"
	const hex = (v: number, skipzero: boolean) => (skipzero && ((v>>4) & 0xf) === 0 ? '' : hexchars[(v>>4) & 0xf]) + hexchars[v&0xf]

	let zeroStart = 0, zeroEnd = 0

	// Find largest run of zeroes.
	let i = 0
	while (i < 16) {
		let j = i
		while (j < 16 && b[j] === 0) {
			j++
		}
		if (j-i > 2 && j-i > zeroEnd-zeroStart) {
			zeroStart = i
			zeroEnd = j
			i = j
		} else if (j > i) {
			i = j
		} else {
			i++
		}
	}

	let s = ''
	for (let i = 0; i < 16; i++) {
		if (i === zeroStart) {
			s += '::'
		} else if (i < zeroStart || i >= zeroEnd) {
			if (i > 0 && i % 2 === 0 && !s.endsWith(':')) {
				s += ':'
			}
			if (i % 2 === 1 || b[i] !== 0) {
				s += hex(b[i], s === '' || s.endsWith(':'))
			}
		}
	}
	return s
}

const ipdomainString = (ipd: api.IPDomain) => {
	if (ipd.IP !== '') {
		return formatIP(ipd.IP)
	}
	return domainString(ipd.Domain)
}

const formatSize = (n: number) => {
	if (n > 10*1024*1024) {
		return Math.round(n/(1024*1024)) + ' mb'
	} else if (n > 500) {
		return Math.round(n/1024) + ' kb'
	}
	return n + ' bytes'
}

const index = async () => {
	const [domains, queueSize, checkUpdatesEnabled] = await Promise.all([
		client.Domains(),
		client.QueueSize(),
		client.CheckUpdatesEnabled(),
	])

	let fieldset: HTMLFieldSetElement
	let domain: HTMLInputElement
	let account: HTMLInputElement
	let localpart: HTMLInputElement

	dom._kids(page,
		crumbs('Mox Admin'),
		checkUpdatesEnabled ? [] : dom.p(box(yellow, 'Warning: Checking for updates has not been enabled in mox.conf (CheckUpdates: true).', dom.br(), 'Make sure you stay up to date through another mechanism!', dom.br(), 'You have a responsibility to keep the internet-connected software you run up to date and secure!', dom.br(), 'See ', link('https://updates.xmox.nl/changelog'))),
		dom.p(
			dom.a('Accounts', attr.href('#accounts')), dom.br(),
			dom.a('Queue', attr.href('#queue')), ' ('+queueSize+')', dom.br(),
		),
		dom.h2('Domains'),
		(domains || []).length === 0 ? box(red, 'No domains') :
		dom.ul(
			(domains || []).map(d => dom.li(dom.a(attr.href('#domains/'+domainName(d)), domainString(d)))),
		),
		dom.br(),
		dom.h2('Add domain'),
		dom.form(
			async function submit(e: SubmitEvent) {
				e.preventDefault()
				e.stopPropagation()
				fieldset.disabled = true
				try {
					await client.DomainAdd(domain.value, account.value, localpart.value)
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldset.disabled = false
				}
				window.location.hash = '#domains/' + domain.value
			},
			fieldset=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					'Domain',
					dom.br(),
					domain=dom.input(attr.required('')),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					'Postmaster/reporting account',
					dom.br(),
					account=dom.input(attr.required('')),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					dom.span('Localpart (optional)', attr.title('Must be set if and only if account does not yet exist. The localpart for the user of this domain. E.g. postmaster.')),
					dom.br(),
					localpart=dom.input(),
				),
				' ',
				dom.submitbutton('Add domain', attr.title('Domain will be added and the config reloaded. You should add the required DNS records after adding the domain.')),
			),
		),
		dom.br(),
		dom.h2('Reports'),
		dom.div(dom.a('DMARC', attr.href('#dmarc/reports'))),
		dom.div(dom.a('TLS', attr.href('#tlsrpt/reports'))),
		dom.br(),
		dom.h2('Operations'),
		dom.div(dom.a('MTA-STS policies', attr.href('#mtasts'))),
		dom.div(dom.a('DMARC evaluations', attr.href('#dmarc/evaluations'))),
		dom.div(dom.a('TLS connection results', attr.href('#tlsrpt/results'))),
		// todo: routing, globally, per domain and per account
		dom.br(),
		dom.h2('DNS blocklist status'),
		dom.div(dom.a('DNSBL status', attr.href('#dnsbl'))),
		dom.br(),
		dom.h2('Configuration'),
		dom.div(dom.a('Webserver', attr.href('#webserver'))),
		dom.div(dom.a('Files', attr.href('#config'))),
		dom.div(dom.a('Log levels', attr.href('#loglevels'))),
		footer,
	)
}

const config = async () => {
	const [staticPath, dynamicPath, staticText, dynamicText] = await client.ConfigFiles()

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'Config',
		),
		dom.h2(staticPath),
		dom.pre(dom._class('literal'), staticText),
		dom.h2(dynamicPath),
		dom.pre(dom._class('literal'), dynamicText),
	)
}

const loglevels = async () => {
	const loglevels = await client.LogLevels()

	const levels = ['error', 'info', 'warn', 'debug', 'trace', 'traceauth', 'tracedata']

	let form: HTMLFormElement
	let fieldset: HTMLFieldSetElement
	let pkg: HTMLInputElement
	let level: HTMLSelectElement

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'Log levels',
		),
		dom.p('Note: changing a log level here only changes it for the current process. When mox restarts, it sets the log levels from the configuration file. Change mox.conf to keep the changes.'),
		dom.table(
			dom.thead(
				dom.tr(
					dom.th('Package', attr.title('Log levels can be configured per package. E.g. smtpserver, imapserver, dkim, dmarc, tlsrpt, etc.')),
					dom.th('Level', attr.title('If you set the log level to "trace", imap and smtp protocol transcripts will be logged. Sensitive authentication is replaced with "***" unless the level is >= "traceauth". Data is masked with "..." unless the level is "tracedata".')),
					dom.th('Action'),
				),
			),
			dom.tbody(
				Object.entries(loglevels).map(t => {
					let lvl: HTMLSelectElement
					return dom.tr(
						dom.td(t[0] || '(default)'),
						dom.td(
							lvl=dom.select(levels.map(l => dom.option(l, t[1] === l ? attr.selected('') : []))),
						),
						dom.td(
							dom.clickbutton('Save', attr.title('Set new log level for package.'), async function click(e: MouseEvent) {
								e.preventDefault()
								const target = e.target! as HTMLButtonElement
								try {
									target.disabled = true
									await client.LogLevelSet(t[0], lvl.value)
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + err)
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: reload just the current loglevels
							}),
							' ',
							dom.clickbutton('Remove', attr.title('Remove this log level, the default log level will apply.'), t[0] === '' ? attr.disabled('') : [], async function click(e: MouseEvent) {
								e.preventDefault()
								const target = e.target! as HTMLButtonElement
								try {
									target.disabled = true
									await client.LogLevelRemove(t[0])
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + err)
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: reload just the current loglevels
							}),
						),
					)
				}),
			),
		),
		dom.br(),
		dom.h2('Add log level setting'),
		form=dom.form(
			async function submit(e: SubmitEvent) {
				e.preventDefault()
				e.stopPropagation()
				fieldset.disabled = true
				try {
					await client.LogLevelSet(pkg.value, level.value)
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldset.disabled = false
				}
				form.reset()
				window.location.reload() // todo: reload just the current loglevels
			},
			fieldset=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					'Package',
					dom.br(),
					pkg=dom.input(attr.required('')),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					'Level',
					dom.br(),
					level=dom.select(
						attr.required(''),
						levels.map(l => dom.option(l, l === 'debug' ? attr.selected('') : [])),
					),
				),
				' ',
				dom.submitbutton('Add'),
			),
			dom.br(),
			dom.p('Suggestions for packages: autotls dkim dmarc dmarcdb dns dnsbl dsn http imapserver iprev junk message metrics mox moxio mtasts mtastsdb publicsuffix queue sendmail serve smtpserver spf store subjectpass tlsrpt tlsrptdb updates'),
		),
	)
}

const box = (color: string, ...l: ElemArg[]) => [
	dom.div(
		style({
			display: 'inline-block',
			padding: '.25em .5em',
			backgroundColor: color,
			borderRadius: '3px',
			margin: '.5ex 0',
		}),
		l,
	),
	dom.br(),
]
const inlineBox = (color: string, ...l: ElemArg[]) =>
	dom.span(
		style({
			display: 'inline-block',
			padding: color ? '0.05em 0.2em' : '',
			backgroundColor: color,
			borderRadius: '3px',
		}),
		l,
	)

const accounts = async () => {
	const accounts = await client.Accounts()

	let fieldset: HTMLFieldSetElement
	let account: HTMLInputElement
	let email: HTMLInputElement

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'Accounts',
		),
		dom.h2('Accounts'),
		(accounts || []).length === 0 ? dom.p('No accounts') :
		dom.ul(
			(accounts || []).map(s => dom.li(dom.a(s, attr.href('#accounts/'+s)))),
		),
		dom.br(),
		dom.h2('Add account'),
		dom.form(
			async function submit(e: SubmitEvent) {
				e.preventDefault()
				e.stopPropagation()
				fieldset.disabled = true
				try {
					await client.AccountAdd(account.value, email.value)
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldset.disabled = false
				}
				window.location.hash = '#accounts/'+account.value
			},
			fieldset=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					'Account name',
					dom.br(),
					account=dom.input(attr.required('')),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					'Email address',
					dom.br(),
					email=dom.input(attr.type('email'), attr.required('')),
				),
				' ',
				dom.submitbutton('Add account', attr.title('The account will be added and the config reloaded.')),
			)
		)
	)
}

const account = async (name: string) => {
	const config = await client.Account(name)

	let form: HTMLFormElement
	let fieldset: HTMLFieldSetElement
	let email: HTMLInputElement

	let fieldsetLimits: HTMLFieldSetElement
	let maxOutgoingMessagesPerDay: HTMLInputElement
	let maxFirstTimeRecipientsPerDay: HTMLInputElement
	let quotaMessageSize: HTMLInputElement

	let formPassword: HTMLFormElement
	let fieldsetPassword: HTMLFieldSetElement
	let password: HTMLInputElement
	let passwordHint: HTMLElement

	const xparseSize = (s: string) => {
		const origs = s
		s = s.toLowerCase()
		let mult = 1
		if (s.endsWith('k')) {
			mult = 1024
		} else if (s.endsWith('m')) {
			mult = 1024*1024
		} else if (s.endsWith('g')) {
			mult = 1024*1024*1024
		} else if (s.endsWith('t')) {
			mult = 1024*1024*1024*1024
		}
		if (mult !== 1) {
			s = s.substring(0, s.length-1)
		}
		let v = parseInt(s)
		console.log('x', s, v, mult, formatQuotaSize(v*mult))
		if (isNaN(v) || origs !== formatQuotaSize(v*mult)) {
			throw new Error('invalid number')
		}
		return v*mult
	}

	const formatQuotaSize = (v: number) => {
		if (v === 0) {
			return '0'
		}
		const m = 1024*1024
		const g = m*1024
		const t = g*1024
		if (Math.floor(v/t)*t === v) {
			return ''+(v/t)+'t'
		} else if (Math.floor(v/g)*g === v) {
			return ''+(v/g)+'g'
		} else if (Math.floor(v/m)*m === v) {
			return ''+(v/m)+'m'
		}
		return ''+v
	}

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('Accounts', '#accounts'),
			name,
		),
		dom.div(
			'Default domain: ',
			config.Domain ? dom.a(config.Domain, attr.href('#domains/'+config.Domain)) : '(none)',
		),
		dom.br(),
		dom.h2('Addresses'),
		dom.table(
			dom.thead(
				dom.tr(
					dom.th('Address'), dom.th('Action'),
				),
			),
			dom.tbody(
				Object.keys(config.Destinations).map(k => {
					let v: ElemArg = k
					const t = k.split('@')
					if (t.length > 1) {
						const d = t[t.length-1]
						const lp = t.slice(0, t.length-1).join('@')
						v = [
							lp, '@',
							dom.a(d, attr.href('#domains/'+d)),
						]
						if (lp === '') {
							v.unshift('(catchall) ')
						}
					}
					return dom.tr(
						dom.td(v),
						dom.td(
							dom.clickbutton('Remove', async function click(e: MouseEvent) {
								e.preventDefault()
								if (!window.confirm('Are you sure you want to remove this address?')) {
									return
								}
								const target = e.target! as HTMLButtonElement
								target.disabled = true
								try {
									let addr = k
									if (!addr.includes('@')) {
										addr += '@' + config.Domain
									}
									await client.AddressRemove(addr)
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + errmsg(err))
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: reload just the list
							}),
						),
					)
				})
			),
		),
		dom.br(),
		dom.h2('Add address'),
		form=dom.form(
			async function submit(e: SubmitEvent) {
				e.preventDefault()
				e.stopPropagation()
				fieldset.disabled = true
				try {
					let addr = email.value
					if (!addr.includes('@')) {
						if (!config.Domain) {
							throw new Error('no default domain configured for account')
						}
						addr += '@' + config.Domain
					}
					await client.AddressAdd(addr, name)
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldset.disabled = false
				}
				form.reset()
				window.location.reload() // todo: only reload the destinations
			},
			fieldset=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					dom.span('Email address or localpart', attr.title('If empty, or localpart is empty, a catchall address is configured for the domain.')),
					dom.br(),
					email=dom.input(),
				),
				' ',
				dom.submitbutton('Add address'),
			),
		),
		dom.br(),
		dom.h2('Limits'),
		dom.form(
			fieldsetLimits=dom.fieldset(
				dom.label(
					style({display: 'block', marginBottom: '.5ex'}),
					dom.span('Maximum outgoing messages per day', attr.title('Maximum number of outgoing messages for this account in a 24 hour window. This limits the damage to recipients and the reputation of this mail server in case of account compromise. Default 1000. MaxOutgoingMessagesPerDay in configuration file.')),
					dom.br(),
					maxOutgoingMessagesPerDay=dom.input(attr.type('number'), attr.required(''), attr.value(config.MaxOutgoingMessagesPerDay || 1000)),
				),
				dom.label(
					style({display: 'block', marginBottom: '.5ex'}),
					dom.span('Maximum first-time recipients per day', attr.title('Maximum number of first-time recipients in outgoing messages for this account in a 24 hour window. This limits the damage to recipients and the reputation of this mail server in case of account compromise. Default 200. MaxFirstTimeRecipientsPerDay in configuration file.')),
					dom.br(),
					maxFirstTimeRecipientsPerDay=dom.input(attr.type('number'), attr.required(''), attr.value(config.MaxFirstTimeRecipientsPerDay || 200)),
				),
				dom.label(
					style({display: 'block', marginBottom: '.5ex'}),
					dom.span('Disk usage quota: Maximum total message size ', attr.title('Default maximum total message size for the account, overriding any globally configured maximum size if non-zero. A negative value can be used to have no limit in case there is a limit by default. Attempting to add new messages beyond the maximum size will result in an error. Useful to prevent a single account from filling storage.')),
					dom.br(),
					quotaMessageSize=dom.input(attr.value(formatQuotaSize(config.QuotaMessageSize))),
				),
				dom.submitbutton('Save'),
			),
			async function submit(e: SubmitEvent) {
				e.stopPropagation()
				e.preventDefault()
				fieldsetLimits.disabled = true
				try {
					await client.SetAccountLimits(name, parseInt(maxOutgoingMessagesPerDay.value) || 0, parseInt(maxFirstTimeRecipientsPerDay.value) || 0, xparseSize(quotaMessageSize.value))
					window.alert('Limits saved.')
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldsetLimits.disabled = false
				}
			},
		),
		dom.br(),
		dom.h2('Set new password'),
		formPassword=dom.form(
			fieldsetPassword=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					'New password',
					dom.br(),
					password=dom.input(attr.type('password'), attr.autocomplete('new-password'), attr.required(''), function focus() {
						passwordHint.style.display = ''
					}),
				),
				' ',
				dom.submitbutton('Change password'),
			),
			passwordHint=dom.div(
				style({display: 'none', marginTop: '.5ex'}),
				dom.clickbutton('Generate random password', function click(e: MouseEvent) {
					e.preventDefault()
					let b = new Uint8Array(1)
					let s = ''
					const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*-_;:,<.>/'
					while (s.length < 12) {
						self.crypto.getRandomValues(b)
						if (Math.ceil(b[0]/chars.length)*chars.length > 255) {
							continue // Prevent bias.
						}
						s += chars[b[0]%chars.length]
					}
					password.type = 'text'
					password.value = s
				}),
				dom.div(dom._class('text'),
					box(yellow, 'Important: Bots will try to bruteforce your password. Connections with failed authentication attempts will be rate limited but attackers WILL find weak passwords. If your account is compromised, spammers are likely to abuse your system, spamming your address and the wider internet in your name. So please pick a random, unguessable password, preferrably at least 12 characters.'),
				),
			),
			async function submit(e: SubmitEvent) {
				e.stopPropagation()
				e.preventDefault()
				fieldsetPassword.disabled = true
				try {
					await client.SetPassword(name, password.value)
					window.alert('Password has been changed.')
					formPassword.reset()
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldsetPassword.disabled = false
				}
			},
		),
		dom.br(),
		dom.h2('Danger'),
		dom.clickbutton('Remove account', async function click(e: MouseEvent) {
			e.preventDefault()
			if (!window.confirm('Are you sure you want to remove this account?')) {
				return
			}
			const target = e.target! as HTMLButtonElement
			target.disabled = true
			try {
				await client.AccountRemove(name)
			} catch (err) {
				console.log({err})
				window.alert('Error: ' + errmsg(err))
				return
			} finally {
				target.disabled = false
			}
			window.location.hash = '#accounts'
		}),
	)
}

const domain = async (d: string) => {
	const end = new Date()
	const start = new Date(new Date().getTime() - 30*24*3600*1000)
	const [dmarcSummaries, tlsrptSummaries, localpartAccounts, dnsdomain, clientConfigs] = await Promise.all([
		client.DMARCSummaries(start, end, d),
		client.TLSRPTSummaries(start, end, d),
		client.DomainLocalparts(d),
		client.Domain(d),
		client.ClientConfigsDomain(d),
	])

	let form: HTMLFormElement
	let fieldset: HTMLFieldSetElement
	let localpart: HTMLInputElement
	let account: HTMLInputElement

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'Domain ' + domainString(dnsdomain),
		),
		dom.ul(
			dom.li(dom.a('Required DNS records', attr.href('#domains/' + d + '/dnsrecords'))),
			dom.li(dom.a('Check current actual DNS records and domain configuration', attr.href('#domains/' + d + '/dnscheck'))),
		),
		dom.br(),
		dom.h2('Client configuration'),
		dom.p('If autoconfig/autodiscover does not work with an email client, use the settings below for this domain. Authenticate with email address and password. ', dom.span('Explicitly configure', attr.title('To prevent authentication mechanism downgrade attempts that may result in clients sending plain text passwords to a MitM.')), ' the first supported authentication mechanism: SCRAM-SHA-256-PLUS, SCRAM-SHA-1-PLUS, SCRAM-SHA-256, SCRAM-SHA-1, CRAM-MD5.'),
		dom.table(
			dom.thead(
				dom.tr(
					dom.th('Protocol'), dom.th('Host'), dom.th('Port'), dom.th('Listener'), dom.th('Note'),
				),
			),
			dom.tbody(
				(clientConfigs.Entries || []).map(e =>
					dom.tr(
						dom.td(e.Protocol),
						dom.td(domainString(e.Host)),
						dom.td(''+e.Port),
						dom.td(''+e.Listener),
						dom.td(''+e.Note),
					)
				),
			),
		),
		dom.br(),
		dom.h2('DMARC aggregate reports summary'),
		renderDMARCSummaries(dmarcSummaries || []),
		dom.br(),
		dom.h2('TLS reports summary'),
		renderTLSRPTSummaries(tlsrptSummaries || []),
		dom.br(),
		dom.h2('Addresses'),
		dom.table(
			dom.thead(
				dom.tr(
					dom.th('Address'), dom.th('Account'), dom.th('Action'),
				),
			),
			dom.tbody(
				Object.entries(localpartAccounts).map(t =>
					dom.tr(
						dom.td(t[0] || '(catchall)'),
						dom.td(dom.a(t[1], attr.href('#accounts/'+t[1]))),
						dom.td(
							dom.clickbutton('Remove', async function click(e: MouseEvent) {
								e.preventDefault()
								if (!window.confirm('Are you sure you want to remove this address?')) {
									return
								}
								const target = e.target! as HTMLButtonElement
								target.disabled = true
								try {
									await client.AddressRemove(t[0] + '@' + d)
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + errmsg(err))
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: only reload the localparts
							}),
						),
					),
				),
			),
		),
		dom.br(),
		dom.h2('Add address'),
		form=dom.form(
			async function submit(e: SubmitEvent) {
				e.preventDefault()
				e.stopPropagation()
				fieldset.disabled = true
				try {
					await client.AddressAdd(localpart.value+'@'+d, account.value)
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldset.disabled = false
				}
				form.reset()
				window.location.reload() // todo: only reload the addresses
			},
			fieldset=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					dom.span('Localpart', attr.title('An empty localpart is the catchall destination/address for the domain.')),
					dom.br(),
					localpart=dom.input(),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					'Account',
					dom.br(),
					account=dom.input(attr.required('')),
				),
				' ',
				dom.submitbutton('Add address', attr.title('Address will be added and the config reloaded.')),
			),
		),
		dom.br(),
		dom.h2('External checks'),
		dom.ul(
			dom.li(link('https://internet.nl/mail/'+dnsdomain.ASCII+'/', 'Check configuration at internet.nl')),
		),
		dom.br(),
		dom.h2('Danger'),
		dom.clickbutton('Remove domain', async function click(e: MouseEvent) {
			e.preventDefault()
			if (!window.confirm('Are you sure you want to remove this domain?')) {
				return
			}
			const target = e.target! as HTMLButtonElement
			target.disabled = true
			try {
				await client.DomainRemove(d)
			} catch (err) {
				console.log({err})
				window.alert('Error: ' + errmsg(err))
				return
			} finally {
				target.disabled = false
			}
			window.location.hash = '#'
		}),
	)
}

const domainDNSRecords = async (d: string) => {
	const [records, dnsdomain] = await Promise.all([
		client.DomainRecords(d),
		client.Domain(d),
	])

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('Domain ' + domainString(dnsdomain), '#domains/'+d),
			'DNS Records',
		),
		dom.h1('Required DNS records'),
		dom.pre('pre', dom._class('literal'), (records || []).join('\n')),
		dom.br(),
	)
}

const domainDNSCheck = async (d: string) => {
	const [checks, dnsdomain] = await Promise.all([
		client.CheckDomain(d),
		client.Domain(d),
	])

	interface Result {
		Errors?: string[] | null
		Warnings?: string[] | null
		Instructions?: string[] | null
	}

	const resultSection = (title: string, r: Result, details: ElemArg[]) => {
		let success: ElemArg[] = []
		if ((r.Errors || []).length === 0 && (r.Warnings || []).length === 0) {
			success = box(green, 'OK')
		}
		const errors = (r.Errors || []).length === 0 ? [] : box(red, dom.ul(style({marginLeft: '1em'}), (r.Errors || []).map(s => dom.li(s))))
		const warnings = (r.Warnings || []).length === 0 ? [] : box(yellow, dom.ul(style({marginLeft: '1em'}), (r.Warnings || []).map(s => dom.li(s))))

		let instructions: HTMLElement | null = null
		if (r.Instructions && r.Instructions.length > 0) {
			instructions = dom.div(style({margin: '.5ex 0'}))
			const instrs = [
				(r.Instructions || []).map(s => [
					dom.pre(dom._class('literal'), style({display: 'inline-block', maxWidth: '60em'}), s),
					dom.br(),
				]),
			]
			if ((r.Errors || []).length === 0) {
				dom._kids(instructions,
					dom.div(
						dom.a('Show instructions', attr.href('#'), function click(e: MouseEvent) {
							e.preventDefault()
							dom._kids(instructions!, instrs)
						}),
						dom.br(),
					)
				)
			} else {
				dom._kids(instructions, instrs)
			}
		}
		return [
			dom.h2(title),
			success,
			errors,
			warnings,
			details,
			dom.br(),
			instructions ? instructions : [],
			dom.br(),
		]
	}

	const detailsDNSSEC: ElemArg[] = []
	const detailsIPRev = !checks.IPRev.IPNames || !Object.entries(checks.IPRev.IPNames).length ? [] : [
		dom.div('Hostname: ' + domainString(checks.IPRev.Hostname)),
		dom.table(
			dom.tr(dom.th('IP'), dom.th('Addresses')),
			Object.entries(checks.IPRev.IPNames).sort().map(t =>
				dom.tr(dom.td(t[0]), dom.td((t[1] || []).join(', '))),
			)
		),
	]
	const detailsMX = (checks.MX.Records || []).length === 0 ? [] : [
		dom.table(
			dom.tr(dom.th('Preference'), dom.th('Host'), dom.th('IPs')),
			(checks.MX.Records || []).map(mx =>
				dom.tr(dom.td(''+mx.Pref), dom.td(mx.Host), dom.td((mx.IPs || []).join(', '))),
			)
		),
	]
	const detailsTLS: ElemArg[] = []
	const detailsDANE: ElemArg[] = []
	const detailsSPF: ElemArg[] = [
		checks.SPF.DomainTXT ? [dom.div('Domain TXT record: ' + checks.SPF.DomainTXT)] : [],
		checks.SPF.HostTXT ? [dom.div('Host TXT record: ' + checks.SPF.HostTXT)] : [],
	]
	const detailsDKIM = (checks.DKIM.Records || []).length === 0 ? [] : [
		dom.table(
			dom.tr(dom.th('Selector'), dom.th('TXT record')),
			(checks.DKIM.Records || []).map(rec =>
				dom.tr(dom.td(rec.Selector), dom.td(rec.TXT)),
			),
		)
	]
	const detailsDMARC = !checks.DMARC.Domain ? [] : [
		dom.div('Domain: ' + checks.DMARC.Domain),
		!checks.DMARC.TXT ? [] : dom.div('TXT record: ' + checks.DMARC.TXT),
	]
	const detailsTLSRPT = (checksTLSRPT: api.TLSRPTCheckResult) => !checksTLSRPT.TXT ? [] : [
		dom.div('TXT record: ' + checksTLSRPT.TXT),
	]
	const detailsMTASTS = !checks.MTASTS.TXT && !checks.MTASTS.PolicyText ? [] : [
		!checks.MTASTS.TXT ? [] : dom.div('MTA-STS record: ' + checks.MTASTS.TXT),
		!checks.MTASTS.PolicyText ? [] : dom.div('MTA-STS policy: ', dom.pre(dom._class('literal'), style({maxWidth: '60em'}), checks.MTASTS.PolicyText)),
	]
	const detailsSRVConf = !checks.SRVConf.SRVs || Object.keys(checks.SRVConf.SRVs).length === 0 ? [] : [
		dom.table(
			dom.tr(dom.th('Service'), dom.th('Priority'), dom.th('Weight'), dom.th('Port'), dom.th('Host')),
			Object.entries(checks.SRVConf.SRVs || []).map(t => {
				const l = t[1]
				if (!l || !l.length) {
					return dom.tr(dom.td(t[0]), dom.td(attr.colspan('4'), '(none)'))
				}
				return l.map(r => dom.tr([t[0], r.Priority, r.Weight, r.Port, r.Target].map(s => dom.td(''+s))))
			}),
		),
	]
	const detailsAutoconf = [
		...(!checks.Autoconf.ClientSettingsDomainIPs ? [] : [dom.div('Client settings domain IPs: ' + checks.Autoconf.ClientSettingsDomainIPs.join(', '))]),
		...(!checks.Autoconf.IPs ? [] : [dom.div('IPs: ' + checks.Autoconf.IPs.join(', '))]),
	]
	const detailsAutodiscover = !checks.Autodiscover.Records ? [] : [
		dom.table(
			dom.tr(dom.th('Host'), dom.th('Port'), dom.th('Priority'), dom.th('Weight'), dom.th('IPs')),
			(checks.Autodiscover.Records || []).map(r =>
				dom.tr([r.Target, r.Port, r.Priority, r.Weight, (r.IPs || []).join(', ')].map(s => dom.td(''+s)))
			),
		),
	]

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('Domain ' + domainString(dnsdomain), '#domains/'+d),
			'Check DNS',
		),
		dom.h1('DNS records and domain configuration check'),
		resultSection('DNSSEC', checks.DNSSEC, detailsDNSSEC),
		resultSection('IPRev', checks.IPRev, detailsIPRev),
		resultSection('MX', checks.MX, detailsMX),
		resultSection('TLS', checks.TLS, detailsTLS),
		resultSection('DANE', checks.DANE, detailsDANE),
		resultSection('SPF', checks.SPF, detailsSPF),
		resultSection('DKIM', checks.DKIM, detailsDKIM),
		resultSection('DMARC', checks.DMARC, detailsDMARC),
		resultSection('Host TLSRPT', checks.HostTLSRPT, detailsTLSRPT(checks.HostTLSRPT)),
		resultSection('Domain TLSRPT', checks.DomainTLSRPT, detailsTLSRPT(checks.DomainTLSRPT)),
		resultSection('MTA-STS', checks.MTASTS, detailsMTASTS),
		resultSection('SRV conf', checks.SRVConf, detailsSRVConf),
		resultSection('Autoconf', checks.Autoconf, detailsAutoconf),
		resultSection('Autodiscover', checks.Autodiscover, detailsAutodiscover),
		dom.br(),
	)
}

const dmarcIndex = async () => {
	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'DMARC',
		),
		dom.ul(
			dom.li(
				dom.a(attr.href('#dmarc/reports'), 'Reports'), ', incoming DMARC aggregate reports.',
			),
			dom.li(
				dom.a(attr.href('#dmarc/evaluations'), 'Evaluations'), ', for outgoing DMARC aggregate reports.',
			),
		),
	)
}

const dmarcReports = async () => {
	const end = new Date()
	const start = new Date(new Date().getTime() - 30*24*3600*1000)
	const summaries = await client.DMARCSummaries(start, end, "")

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('DMARC', '#dmarc'),
			'Aggregate reporting summary',
		),
		dom.p('DMARC reports are periodically sent by other mail servers that received an email message with a "From" header with our domain. Domains can have a DMARC DNS record that asks other mail servers to send these aggregate reports for analysis.'),
		renderDMARCSummaries(summaries || []),
	)
}

const renderDMARCSummaries = (summaries: api.DMARCSummary[]) => {
	return [
		dom.p('Below a summary of DMARC aggregate reporting results for the past 30 days.'),
		summaries.length === 0 ? dom.div(box(yellow, 'No domains with reports.')) :
		dom.table(
			dom.thead(
				dom.tr(
					dom.th('Domain', attr.title('Domain to which the DMARC policy applied. If example.com has a DMARC policy, and email is sent with a From-header with subdomain.example.com, and there is no DMARC record for that subdomain, but there is one for example.com, then the DMARC policy of example.com applies and reports are sent for that that domain.')),
					dom.th('Messages', attr.title('Total number of messages that had the DMARC policy applied and reported. Actual messages sent is likely higher because not all email servers send DMARC aggregate reports, or perform DMARC checks at all.')),
					dom.th('DMARC "quarantine"/"reject"', attr.title('Messages for which policy was to mark them as spam (quarantine) or reject them during SMTP delivery.')),
					dom.th('DKIM "fail"', attr.title('Messages with a failing DKIM check. This can happen when sending through a mailing list where that list keeps your address in the message From-header but also strips DKIM-Signature headers in the message. DMARC evaluation passes if either DKIM passes or SPF passes.')),
					dom.th('SPF "fail"', attr.title('Message with a failing SPF check. This can happen with email forwarding and with mailing list. Other mail servers have sent email with this domain in the message From-header. DMARC evaluation passes if at least SPF or DKIM passes.')),
					dom.th('Policy overrides', attr.title('Mail servers can override the DMARC policy. E.g. a mail server may be able to detect emails coming from mailing lists that do not pass DMARC and would have to be rejected, but for which an override has been configured.')),
				)
			),
			dom.tbody(
				summaries.map(r =>
					dom.tr(
						dom.td(dom.a(attr.href('#domains/' + r.Domain + '/dmarc'), attr.title('See report details.'), r.Domain)),
						dom.td(style({textAlign: 'right'}), '' + r.Total),
						dom.td(style({textAlign: 'right'}), r.DispositionQuarantine === 0 && r.DispositionReject === 0 ? '0/0' : box(red, '' + r.DispositionQuarantine + '/' + r.DispositionReject)),
						dom.td(style({textAlign: 'right'}), box(r.DKIMFail === 0 ? green : red, '' + r.DKIMFail)),
						dom.td(style({textAlign: 'right'}), box(r.SPFFail === 0 ? green : red, '' + r.SPFFail)),
						dom.td(!r.PolicyOverrides ? [] : Object.entries(r.PolicyOverrides).map(kv => kv[0] + ': ' + kv[1]).join('; ')),
					)
				),
			),
		)
	]
}

const dmarcEvaluations = async () => {
	const [evalStats, suppressAddresses] = await Promise.all([
		client.DMARCEvaluationStats(),
		client.DMARCSuppressList(),
	])

	const isEmpty = <T>(o: { [key: string]: T }) => {
		for (const _ in o) {
			return false
		}
		return true
	}

	let fieldset: HTMLFieldSetElement
	let reportingAddress: HTMLInputElement
	let until: HTMLInputElement
	let comment: HTMLInputElement

	const nextmonth = new Date(new Date().getTime()+31*24*3600*1000)

	dom._kids(page,
		crumbs(
			crumblink('Beacon Admin', '#'),
			crumblink('DMARC', '#dmarc'),
			'Evaluations',
		),
		dom.p('Incoming messages are checked against the DMARC policy of the domain in the message From header. If the policy requests reporting on the resulting evaluations, they are stored in the database. Each interval of 1 to 24 hours, the evaluations may be sent to a reporting address specified in the domain\'s DMARC policy. Not all evaluations are a reason to send a report, but if a report is sent all evaluations are included.'),
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('Domain', attr.title('Domain in the message From header. Keep in mind these can be forged, so this does not necessarily mean someone from this domain authentically tried delivering email.')),
					dom.th('Dispositions', attr.title('Unique dispositions occurring in report.')),
					dom.th('Evaluations', attr.title('Total number of message delivery attempts, including retries.')),
					dom.th('Send report', attr.title('Whether the current evaluations will cause a report to be sent.')),
				),
			),
			dom.tbody(
				Object.entries(evalStats).sort((a, b) => a[0] < b[0] ? -1 : 1).map(t =>
					dom.tr(
						dom.td(dom.a(attr.href('#dmarc/evaluations/'+domainName(t[1].Domain)), domainString(t[1].Domain))),
						dom.td((t[1].Dispositions || []).join(' ')),
						dom.td(style({textAlign: 'right'}), ''+t[1].Count),
						dom.td(style({textAlign: 'right'}), t[1].SendReport ? '✓' : ''),
					),
				),
				isEmpty(evalStats) ? dom.tr(dom.td(attr.colspan('3'), 'No evaluations.')) : [],
			),
		),
		dom.br(),
		dom.br(),
		dom.h2('Suppressed reporting addresses'),
		dom.p('In practice, sending a DMARC report to a reporting address can cause DSN to be sent back. Such addresses can be added to a supression list for a period, to reduce noise in the postmaster mailbox.'),
		dom.form(
			async function submit(e: SubmitEvent) {
				e.stopPropagation()
				e.preventDefault()
				try {
					fieldset.disabled = true
					await client.DMARCSuppressAdd(reportingAddress.value, new Date(until.value), comment.value)
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldset.disabled = false
				}
				window.location.reload() // todo: add the address to the list, or only reload the list
			},
			fieldset=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					'Reporting address',
					dom.br(),
					reportingAddress=dom.input(attr.required('')),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					'Until',
					dom.br(),
					until=dom.input(attr.type('date'), attr.required(''), attr.value(nextmonth.getFullYear()+'-'+(1+nextmonth.getMonth())+'-'+nextmonth.getDate())),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					dom.span('Comment (optional)'),
					dom.br(),
					comment=dom.input(),
				),
				' ',
				dom.submitbutton('Add', attr.title('Outgoing reports to this reporting address will be suppressed until the end time.')),
			),
		),
		dom.br(),
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('Reporting address'),
					dom.th('Until'),
					dom.th('Comment'),
					dom.th('Action'),
				),
			),
			dom.tbody(
				(suppressAddresses || []).length === 0 ? dom.tr(dom.td(attr.colspan('4'), 'No suppressed reporting addresses.')) : [],
				(suppressAddresses || []).map(ba =>
					dom.tr(
						dom.td(ba.ReportingAddress),
						dom.td(ba.Until.toISOString()),
						dom.td(ba.Comment),
						dom.td(
							dom.clickbutton('Remove', async function click(e: MouseEvent) {
								const target = e.target! as HTMLButtonElement
								try {
									target.disabled = true
									await client.DMARCSuppressRemove(ba.ID)
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + errmsg(err))
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: only reload the list
							}),
							' ',
							dom.clickbutton('Extend for 1 month', async function click(e: MouseEvent) {
								const target = e.target! as HTMLButtonElement
								try {
									target.disabled = true
									await client.DMARCSuppressExtend(ba.ID, new Date(new Date().getTime() + 31*24*3600*1000))
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + errmsg(err))
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: only reload the list
							}),
						),
					)
				),
			),
		),
	)
}

const dmarcEvaluationsDomain = async (domain: string) => {
	const [d, evaluations] = await client.DMARCEvaluationsDomain(domain)

	let lastInterval = ''
	let lastAddresses = ''

	const formatPolicy = (e: api.Evaluation) => {
		const p = e.PolicyPublished
		let s = ''
		const add = (k: string, v: string) => {
			if (v) {
				s += k+'='+v+'; '
			}
		}
		add('p', p.Policy)
		add('sp', p.SubdomainPolicy)
		add('adkim', p.ADKIM)
		add('aspf', p.ASPF)
		add('pct', ''+p.Percentage)
		add('fo', ''+p.ReportingOptions)
		return s
	}
	let lastPolicy = ''

	const authStatus = (v: boolean) => inlineBox(v ? '' : yellow, v ? 'pass' : 'fail')
	const formatDKIMResults = (results: api.DKIMAuthResult[]) => results.map(r => dom.div('selector '+r.Selector+(r.Domain !== domain ? ', domain '+r.Domain : '') + ': ', inlineBox(r.Result === "pass" ? '' : yellow, r.Result)))
	const formatSPFResults = (alignedpass: boolean, results: api.SPFAuthResult[]) => results.map(r => dom.div(''+r.Scope+(r.Domain !== domain ? ', domain '+r.Domain : '') + ': ', inlineBox(r.Result === "pass" && alignedpass ? '' : yellow, r.Result)))

	const sourceIP = (ip: string) => {
		const r = dom.span(ip, attr.title('Click to do a reverse lookup of the IP.'), style({cursor: 'pointer'}), async function click(e: MouseEvent) {
			e.preventDefault()
			try {
				const rev = await client.LookupIP(ip)
				r.innerText = ip + '\n' + (rev.Hostnames || []).join('\n')
			} catch (err) {
				r.innerText = ip + '\nerror: ' +errmsg(err)
			}
		})
		return r
	}

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('DMARC', '#dmarc'),
			crumblink('Evaluations', '#dmarc/evaluations'),
			'Domain '+domainString(d),
		),
		dom.div(
			dom.clickbutton('Remove evaluations', async function click(e: MouseEvent) {
				const target = e.target! as HTMLButtonElement
				target.disabled = true
				try {
					await client.DMARCRemoveEvaluations(domain)
					window.location.reload() // todo: only clear the table?
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
				} finally {
					target.disabled = false
				}
			}),
		),
		dom.br(),
		dom.p('The evaluations below will be sent in a DMARC aggregate report to the addresses found in the published DMARC DNS record, which is fetched again before sending the report. The fields Interval hours, Addresses and Policy are only filled for the first row and whenever a new value in the published DMARC record is encountered.'),
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('ID'),
					dom.th('Evaluated'),
					dom.th('Optional', attr.title('Some evaluations will not cause a DMARC aggregate report to be sent. But if a report is sent, optional records are included.')),
					dom.th('Interval hours', attr.title('DMARC policies published by a domain can specify how often they would like to receive reports. The default is 24 hours, but can be as often as each hour. To keep reports comparable between different mail servers that send reports, reports are sent at rounded up intervals of whole hours that can divide a 24 hour day, and are aligned with the start of a day at UTC.')),
					dom.th('Addresses', attr.title('Addresses that will receive the report. An address can have a maximum report size configured. If there is no address, no report will be sent.')),
					dom.th('Policy', attr.title('Summary of the policy as encountered in the DMARC DNS record of the domain, and used for evaluation.')),
					dom.th('IP', attr.title('IP address of delivery attempt that was evaluated, relevant for SPF.')),
					dom.th('Disposition', attr.title('Our decision to accept/reject this message. It may be different than requested by the published policy. For example, when overriding due to delivery from a mailing list or forwarded address.')),
					dom.th('Aligned DKIM/SPF', attr.title('Whether DKIM and SPF had an aligned pass, where strict/relaxed alignment means whether the domain of an SPF pass and DKIM pass matches the exact domain (strict) or optionally a subdomain (relaxed). A DMARC pass requires at least one pass.')),
					dom.th('Envelope to', attr.title('Domain used in SMTP RCPT TO during delivery.')),
					dom.th('Envelope from', attr.title('Domain used in SMTP MAIL FROM during delivery.')),
					dom.th('Message from', attr.title('Domain in "From" message header.')),
					dom.th('DKIM details', attr.title('Results of verifying DKIM-Signature headers in message. Only signatures with matching organizational domain are included, regardless of strict/relaxed DKIM alignment in DMARC policy.')),
					dom.th('SPF details', attr.title('Results of SPF check used in DMARC evaluation. "mfrom" indicates the "SMTP MAIL FROM" domain was used, "helo" indicates the SMTP EHLO domain was used.')),
				),
			),
			dom.tbody(
				(evaluations || []).map(e => {
					const ival = e.IntervalHours + 'h'
					const interval = ival === lastInterval ? '' : ival
					lastInterval = ival

					const a = (e.Addresses || []).join('\n')
					const addresses = a === lastAddresses ? '' : a
					lastAddresses = a

					const p = formatPolicy(e)
					const policy = p === lastPolicy ? '' : p
					lastPolicy = p

					return dom.tr(
						dom.td(''+e.ID),
						dom.td(new Date(e.Evaluated).toUTCString()),
						dom.td(e.Optional ? 'Yes' : ''),
						dom.td(interval),
						dom.td(addresses),
						dom.td(policy),
						dom.td(sourceIP(e.SourceIP)),
						dom.td(inlineBox(e.Disposition === 'none' ? '' : red, e.Disposition), (e.OverrideReasons || []).length > 0 ? ' ('+(e.OverrideReasons || []).map(r => r.Type).join(', ')+')' : ''),
						dom.td(authStatus(e.AlignedDKIMPass), '/', authStatus(e.AlignedSPFPass)),
						dom.td(e.EnvelopeTo),
						dom.td(e.EnvelopeFrom),
						dom.td(e.HeaderFrom),
						dom.td(formatDKIMResults(e.DKIMResults || [])),
						dom.td(formatSPFResults(e.AlignedSPFPass, e.SPFResults || [])),
					)
				}),
				(evaluations || []).length === 0 ? dom.tr(dom.td(attr.colspan('14'), 'No evaluations.')) : [],
			),
		),
	)
}

const utcDate = (dt: Date) => new Date(Date.UTC(dt.getUTCFullYear(), dt.getUTCMonth(), dt.getUTCDate(), dt.getUTCHours(), dt.getUTCMinutes(), dt.getUTCSeconds()))
const utcDateStr = (dt: Date) => [dt.getUTCFullYear(), 1+dt.getUTCMonth(), dt.getUTCDate()].join('-')
const isDayChange = (dt: Date) => utcDateStr(new Date(dt.getTime() - 2*60*1000)) !== utcDateStr(new Date(dt.getTime() + 2*60*1000))

const period = (start: Date, end: Date) => {
	const beginUTC = utcDate(start)
	const endUTC = utcDate(end)
	const beginDayChange = isDayChange(beginUTC)
	const endDayChange = isDayChange(endUTC)
	let beginstr = utcDateStr(beginUTC)
	let endstr = utcDateStr(endUTC)
	const title = attr.title('' + beginUTC.toISOString() + ' - ' + endUTC.toISOString())
	if (beginDayChange && endDayChange && Math.abs(beginUTC.getTime() - endUTC.getTime()) < 24*(2*60+3600)*1000) {
		return dom.span(beginstr, title)
	}
	const pad = (v: number) => v < 10 ? '0'+v : ''+v
	if (!beginDayChange) {
		beginstr += ' '+pad(beginUTC.getUTCHours()) + ':' + pad(beginUTC.getUTCMinutes())
	}
	if (!endDayChange) {
		endstr += ' '+pad(endUTC.getUTCHours()) + ':' + pad(endUTC.getUTCMinutes())
	}
	return dom.span(beginstr + ' - ' + endstr, title)
}

const domainDMARC = async (d: string) => {
	const end = new Date()
	const start = new Date(new Date().getTime() - 30*24*3600*1000)
	const [reports, dnsdomain] = await Promise.all([
		client.DMARCReports(start, end, d),
		client.Domain(d),
	])

	// todo future: table sorting? period selection (last day, 7 days, 1 month, 1 year, custom period)? collapse rows for a report? show totals per report? a simple bar graph to visualize messages and dmarc/dkim/spf fails? similar for TLSRPT.

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('Domain ' + domainString(dnsdomain), '#domains/'+d),
			'DMARC aggregate reports',
		),
		dom.p('DMARC reports are periodically sent by other mail servers that received an email message with a "From" header with our domain. Domains can have a DMARC DNS record that asks other mail servers to send these aggregate reports for analysis.'),
		dom.p('Below the DMARC aggregate reports for the past 30 days.'),
		(reports || []).length === 0 ? dom.div('No DMARC reports for domain.') :
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('ID'),
					dom.th('Organisation', attr.title('Organization that sent the DMARC report.')),
					dom.th('Period (UTC)', attr.title('Period this reporting period is about. Mail servers are recommended to stick to whole UTC days.')),
					dom.th('Policy', attr.title('The DMARC policy that the remote mail server had fetched and applied to the message. A policy that changed during the reporting period may result in unexpected policy evaluations.')),
					dom.th('Source IP', attr.title('Remote IP address of session at remote mail server.')),
					dom.th('Messages', attr.title('Total messages that the results apply to.')),
					dom.th('Result', attr.title('DMARC evaluation result.')),
					dom.th('ADKIM', attr.title('DKIM alignment. For a pass, one of the DKIM signatures that pass must be strict/relaxed-aligned with the domain, as specified by the policy.')),
					dom.th('ASPF', attr.title('SPF alignment. For a pass, the SPF policy must pass and be strict/relaxed-aligned with the domain, as specified by the policy.')),
					dom.th('SMTP to', attr.title('Domain of destination address, as specified during the SMTP session.')),
					dom.th('SMTP from', attr.title('Domain of originating address, as specified during the SMTP session.')),
					dom.th('Header from', attr.title('Domain of address in From-header of message.')),
					dom.th('Auth Results', attr.title('Details of DKIM and/or SPF authentication results. DMARC requires at least one aligned DKIM or SPF pass.')),
				),
			),
			dom.tbody(
				(reports || []).map(r => {
					const m = r.ReportMetadata

					let policy: string[] = []
					if (r.PolicyPublished.Domain !== d) {
						policy.push(r.PolicyPublished.Domain)
					}
					const alignments = {'r': 'relaxed', 's': 'strict'}
					if (r.PolicyPublished.ADKIM as string !== '') {
						policy.push('dkim '+(alignments[r.PolicyPublished.ADKIM] || r.PolicyPublished.ADKIM))
					}
					if (r.PolicyPublished.ASPF as string !== '') {
						policy.push('spf '+(alignments[r.PolicyPublished.ASPF] || r.PolicyPublished.ASPF))
					}
					if (r.PolicyPublished.Policy as string !== '') {
						policy.push('policy '+r.PolicyPublished.Policy)
					}
					if (r.PolicyPublished.SubdomainPolicy as string !== '' && r.PolicyPublished.SubdomainPolicy !== r.PolicyPublished.Policy) {
						policy.push('subdomain '+r.PolicyPublished.SubdomainPolicy)
					}
					if (r.PolicyPublished.Percentage !== 100) {
						policy.push('' + r.PolicyPublished.Percentage + '%')
					}

					const sourceIP = (ip: string) => {
						const r = dom.span(ip, attr.title('Click to do a reverse lookup of the IP.'), style({cursor: 'pointer'}), async function click(e: MouseEvent) {
							e.preventDefault()
							try {
								const rev = await client.LookupIP(ip)
								r.innerText = ip + '\n' + (rev.Hostnames || []).join('\n')
							} catch (err) {
								r.innerText = ip + '\nerror: ' +errmsg(err)
							}
						})
						return r
					}

					let authResults = 0
					for (const record of (r.Records || [])) {
						authResults += (record.AuthResults.DKIM || []).length
						authResults += (record.AuthResults.SPF || []).length
					}
					const reportRowspan = attr.rowspan('' + authResults)
					return (r.Records || []).map((record, recordIndex) => {
						const row = record.Row
						const pol = row.PolicyEvaluated
						const ids = record.Identifiers
						const dkims = record.AuthResults.DKIM || []
						const spfs = record.AuthResults.SPF || []

						const recordRowspan = attr.rowspan('' + (dkims.length+spfs.length))
						const valignTop = style({verticalAlign: 'top'})

						const dmarcStatuses = {
							none: 'DMARC checks or were not applied. This does not mean these messages are definitely not spam though, and they may have been rejected based on other checks, such as reputation or content-based filters.',
							quarantine: 'DMARC policy is to mark message as spam.',
							reject: 'DMARC policy is to reject the message during SMTP delivery.',
						}
						const rows: HTMLElement[] = []
						const addRow = (...last: ElemArg[]) => {
							const tr = dom.tr(
								recordIndex > 0 || rows.length > 0 ? [] : [
									dom.td(reportRowspan, valignTop, dom.a('' + r.ID, attr.href('#domains/' + d + '/dmarc/' + r.ID), attr.title('View raw report.'))),
									dom.td(reportRowspan, valignTop, m.OrgName, attr.title('Email: ' + m.Email + ', ReportID: ' + m.ReportID)),
									dom.td(reportRowspan, valignTop, period(new Date(m.DateRange.Begin*1000), new Date(m.DateRange.End*1000)), m.Errors && m.Errors.length ? dom.span('errors', attr.title(m.Errors.join('; '))) : []),
									dom.td(reportRowspan, valignTop, policy.join(', ')),
								],
								rows.length > 0 ? [] : [
									dom.td(recordRowspan, valignTop, sourceIP(row.SourceIP)),
									dom.td(recordRowspan, valignTop, '' + row.Count),
									dom.td(recordRowspan, valignTop,
										dom.span(pol.Disposition === 'none' ? 'none' : box(red, pol.Disposition), attr.title(pol.Disposition + ': ' + dmarcStatuses[pol.Disposition])),
										(pol.Reasons || []).map(reason => [dom.br(), dom.span(reason.Type + (reason.Comment ? ' (' + reason.Comment + ')' : ''), attr.title('Policy was overridden by remote mail server for this reasons.'))]),
									),
									dom.td(recordRowspan, valignTop, pol.DKIM === 'pass' ? 'pass' : box(yellow, dom.span(pol.DKIM, attr.title('No or no valid DKIM-signature is present that is "aligned" with the domain name.')))),
									dom.td(recordRowspan, valignTop, pol.SPF === 'pass' ? 'pass' : box(yellow, dom.span(pol.SPF, attr.title('No SPF policy was found, or IP is not allowed by policy, or domain name is not "aligned" with the domain name.')))),
									dom.td(recordRowspan, valignTop, ids.EnvelopeTo),
									dom.td(recordRowspan, valignTop, ids.EnvelopeFrom),
									dom.td(recordRowspan, valignTop, ids.HeaderFrom),
								],
								dom.td(last),
							)
							rows.push(tr)
						}
						for (const dkim of dkims) {
							const statuses = {
								none: 'Message was not signed',
								pass: 'Message was signed and signature was verified.',
								fail: 'Message was signed, but signature was invalid.',
								policy: 'Message was signed, but signature is not accepted by policy.',
								neutral: 'Message was signed, but the signature contains an error or could not be processed. This status is also used for errors not covered by other statuses.',
								temperror: 'Message could not be verified. E.g. because of DNS resolve error. A later attempt may succeed. A missing DNS record is treated as temporary error, a new key may not have propagated through DNS shortly after it was taken into use.',
								permerror: 'Message cannot be verified. E.g. when a required header field is absent or for invalid (combination of) parameters. We typically set this if a DNS record does not allow the signature, e.g. due to algorithm mismatch or expiry.',
							}
							addRow(
								'dkim: ',
								dom.span((dkim.Result === 'none' || dkim.Result === 'pass') ? dkim.Result : box(yellow, dkim.Result), attr.title((dkim.HumanResult ? 'additional information: ' + dkim.HumanResult + ';\n' : '') + dkim.Result + ': ' + (statuses[dkim.Result] || 'invalid status'))),
								!dkim.Selector ? [] : [
									', ',
									dom.span(dkim.Selector, attr.title('Selector, the DKIM record is at "<selector>._domainkey.<domain>".' + (dkim.Domain === d ? '' : ';\ndomain: ' + dkim.Domain))),
								]
							)
						}
						for (const spf of spfs) {
							const statuses = {
								none: 'No SPF policy found.',
								neutral: 'Policy states nothing about IP, typically due to "?" qualifier in SPF record.',
								pass: 'IP is authorized.',
								fail: 'IP is explicitly not authorized, due to "-" qualifier in SPF record.',
								softfail: 'Weak statement that IP is probably not authorized, "~" qualifier in SPF record.',
								temperror: 'Trying again later may succeed, e.g. for temporary DNS lookup error.',
								permerror: 'Error requiring some intervention to correct. E.g. invalid DNS record.',
							}
							addRow(
								'spf: ',
								dom.span((spf.Result === 'none' || spf.Result === 'neutral' || spf.Result === 'pass') ? spf.Result : box(yellow, spf.Result), attr.title(spf.Result + ': ' + (statuses[spf.Result] || 'invalid status'))),
								', ',
								dom.span(spf.Scope, attr.title('scopes:\nhelo: "SMTP HELO"\nmfrom: SMTP "MAIL FROM"')),
								' ',
								dom.span(spf.Domain),
							)
						}
						return rows
					})
				}),
			),
		)
	)
}

const domainDMARCReport = async (d: string, reportID: number) => {
	const [report, dnsdomain] = await Promise.all([
		client.DMARCReportID(d, reportID),
		client.Domain(d),
	])

	dom._kids(page,
		crumbs(
			crumblink('Beacon Admin', '#'),
			crumblink('Domain ' + domainString(dnsdomain), '#domains/'+d),
			crumblink('DMARC aggregate reports', '#domains/' + d + '/dmarc'),
			'Report ' + reportID
		),
		dom.p('Below is the raw report as received from the remote mail server.'),
		dom.div(dom._class('literal'), JSON.stringify(report, null, '\t')),
	)
}

const tlsrptIndex = async () => {
	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'TLSRPT',
		),
		dom.ul(
			dom.li(
				dom.a(attr.href('#tlsrpt/reports'), 'Reports'), ', incoming TLS reports.',
			),
			dom.li(
				dom.a(attr.href('#tlsrpt/results'), 'Results'), ', for outgoing TLS reports.',
			),
		),
	)
}

const tlsrptResults = async () => {
	const [results, suppressAddresses] = await Promise.all([
		client.TLSRPTResults(),
		client.TLSRPTSuppressList(),
	])

	// todo: add a view where results are grouped by policy domain+dayutc. now each recipient domain gets a row.

	let fieldset: HTMLFieldSetElement
	let reportingAddress: HTMLInputElement
	let until: HTMLInputElement
	let comment: HTMLInputElement
	const nextmonth = new Date(new Date().getTime()+31*24*3600*1000)

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('TLSRPT', '#tlsrpt'),
			'Results',
		),
		dom.p('Messages are delivered with SMTP with TLS using STARTTLS if supported and/or required by the recipient domain\'s mail server. TLS connections may fail for various reasons, such as mismatching certificate host name, expired certificates or TLS protocol version/cipher suite incompatibilities. Statistics about successful connections and failed connections are tracked. Results can be tracked for recipient domains (for MTA-STS policies), and per MX host (for DANE). A domain/host can publish a TLSRPT DNS record with addresses that should receive TLS reports. Reports are sent every 24 hours. Not all results are enough reason to send a report, but if a report is sent all results are included. By default, reports are only sent if a report contains a connection failure. Sending reports about all-successful connections can be configured. Reports sent to recipient domains include the results for its MX hosts, and reports for an MX host reference the recipient domains.'),
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('Day (UTC)', attr.title('Day covering these results, a whole day from 00:00 UTC to 24:00 UTC.')),
					dom.th('Recipient domain', attr.title('Domain of addressee. For delivery to a recipient, the recipient and policy domains will match for reporting on MTA-STS policies, but can also result in reports for hosts from the MX record of the recipient to report on DANE policies.')),
					dom.th('Policy domain', attr.title('Domain for TLSRPT policy, specifying URIs to which reports should be sent.')),
					dom.th('Host', attr.title('Whether policy domain is an (MX) host (for DANE), or a recipient domain (for MTA-STS).')),
					dom.th('Policies', attr.title('Policies found.')),
					dom.th('Success', attr.title('Total number of successful connections.')),
					dom.th('Failure', attr.title('Total number of failed connection attempts.')),
					dom.th('Failure details', attr.title('Total number of details about failures.')),
					dom.th('Send report', attr.title('Whether the current results may cause a report to be sent. To prevent report loops, reports are not sent for TLS connections used to deliver TLS or DMARC reports. Whether a report is eventually sent depends on more factors, such as whether the policy domain has a TLSRPT policy with reporting addresses, and whether TLS connection failures were registered (depending on configuration).')),
				),
			),
			dom.tbody(
				(results || []).sort((a, b) => {
					if (a.DayUTC !== b.DayUTC) {
						return a.DayUTC < b.DayUTC ? -1 : 1
					}
					if (a.RecipientDomain !== b.RecipientDomain) {
						return a.RecipientDomain < b.RecipientDomain ? -1 : 1
					}
					return a.PolicyDomain < b.PolicyDomain ? -1 : 1
				}).map(r => {
					let success = 0
					let failed = 0
					let failureDetails = 0
					;(r.Results || []).forEach(result => {
						success += result.Summary.TotalSuccessfulSessionCount
						failed += result.Summary.TotalFailureSessionCount
						failureDetails += (result.FailureDetails || []).length
					})
					const policyTypes: string[] = []
					for (const result of (r.Results || [])) {
						const pt = result.Policy.Type
						if (!policyTypes.includes(pt)) {
							policyTypes.push(pt)
						}
					}
					return dom.tr(
						dom.td(r.DayUTC),
						dom.td(r.RecipientDomain),
						dom.td(dom.a(attr.href('#tlsrpt/results/'+ (r.RecipientDomain === r.PolicyDomain ? 'rcptdom/' : 'host/') + r.PolicyDomain), r.PolicyDomain)),
						dom.td(r.IsHost ? '✓' : ''),
						dom.td(policyTypes.join(', ')),
						dom.td(style({textAlign: 'right'}), ''+success),
						dom.td(style({textAlign: 'right'}), ''+failed),
						dom.td(style({textAlign: 'right'}), ''+failureDetails),
						dom.td(style({textAlign: 'right'}), r.SendReport ? '✓' : ''),
					)
				}),
				(results || []).length === 0 ? dom.tr(dom.td(attr.colspan('9'), 'No results.')) : [],
			),
		),
		dom.br(),
		dom.br(),
		dom.h2('Suppressed reporting addresses'),
		dom.p('In practice, sending a TLS report to a reporting address can cause DSN to be sent back. Such addresses can be added to a suppress list for a period, to reduce noise in the postmaster mailbox.'),
		dom.form(
			async function submit(e: SubmitEvent) {
				e.stopPropagation()
				e.preventDefault()
				try {
					fieldset.disabled = true
					await client.TLSRPTSuppressAdd(reportingAddress.value, new Date(until.value), comment.value)
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
					return
				} finally {
					fieldset.disabled = false
				}
				window.location.reload() // todo: add the address to the list, or only reload the list
			},
			fieldset=dom.fieldset(
				dom.label(
					style({display: 'inline-block'}),
					'Reporting address',
					dom.br(),
					reportingAddress=dom.input(attr.required('')),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					'Until',
					dom.br(),
					until=dom.input(attr.type('date'), attr.required(''), attr.value(nextmonth.getFullYear()+'-'+(1+nextmonth.getMonth())+'-'+nextmonth.getDate())),
				),
				' ',
				dom.label(
					style({display: 'inline-block'}),
					dom.span('Comment (optional)'),
					dom.br(),
					comment=dom.input(),
				),
				' ',
				dom.submitbutton('Add', attr.title('Outgoing reports to this reporting address will be suppressed until the end time.')),
			),
		),
		dom.br(),
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('Reporting address'),
					dom.th('Until'),
					dom.th('Comment'),
					dom.th('Action'),
				),
			),
			dom.tbody(
				(suppressAddresses || []).length === 0 ? dom.tr(dom.td(attr.colspan('4'), 'No suppressed reporting addresses.')) : [],
				(suppressAddresses || []).map(ba =>
					dom.tr(
						dom.td(ba.ReportingAddress),
						dom.td(ba.Until.toISOString()),
						dom.td(ba.Comment),
						dom.td(
							dom.clickbutton('Remove', async function click(e: MouseEvent) {
								const target = e.target! as HTMLButtonElement
								try {
									target.disabled = true
									await client.TLSRPTSuppressRemove(ba.ID)
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + errmsg(err))
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: only reload the list
							}),
							' ',
							dom.clickbutton('Extend for 1 month', async function click(e: MouseEvent) {
								const target = e.target! as HTMLButtonElement
								try {
									target.disabled = true
									await client.TLSRPTSuppressExtend(ba.ID, new Date(new Date().getTime() + 31*24*3600*1000))
								} catch (err) {
									console.log({err})
									window.alert('Error: ' + errmsg(err))
									return
								} finally {
									target.disabled = false
								}
								window.location.reload() // todo: only reload the list
							}),
						),
					)
				),
			),
		),
	)
}

const tlsrptResultsPolicyDomain = async (isrcptdom: boolean, domain: string) => {
	const [d, tlsresults] = await client.TLSRPTResultsDomain(isrcptdom, domain)
	const recordPromise = client.LookupTLSRPTRecord(domain)

	let recordBox: HTMLElement
	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('TLSRPT', '#tlsrpt'),
			crumblink('Results', '#tlsrpt/results'),
			(isrcptdom ? 'Recipient domain ' : 'Host ') + domainString(d),
		),
		dom.div(
			dom.clickbutton('Remove results', async function click(e: MouseEvent) {
				e.preventDefault()
				const target = e.target! as HTMLButtonElement
				target.disabled = true
				try {
					await client.TLSRPTRemoveResults(isrcptdom, domain, '')
					window.location.reload() // todo: only clear the table?
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
				} finally {
					target.disabled = false
				}
			}),
		),
		dom.br(),
		dom.div('Fetching TLSRPT DNS record...'),
		recordBox=dom.div(),
		dom.br(),
		dom.p('Below are the results per day and ' + (isrcptdom ? 'policy' : 'recipient') + ' domain that may be sent in a report.'),
		(tlsresults || []).map(tlsresult => [
			dom.h2(tlsresult.DayUTC, ' - ', dom.span(attr.title('Recipient domain, as used in SMTP MAIL TO, usually based on message To/Cc/Bcc.'), isrcptdom ? tlsresult.PolicyDomain : tlsresult.RecipientDomain)),
			dom.p(
				'Send report (if TLSRPT policy exists and has address): '+(tlsresult.SendReport ? 'Yes' : 'No'),
				dom.br(),
				'Report about (MX) host (instead of recipient domain): '+(tlsresult.IsHost ? 'Yes' : 'No'),
			),
			dom.div(dom._class('literal'), JSON.stringify(tlsresult.Results, null, '\t')),
		])
	)

	// In background so page load fade doesn't look weird.
	;(async () => {
		let txt: string = ''
		let error: string
		try {
			let [_, xtxt, xerror] = await recordPromise
			txt = xtxt
			error = xerror
		} catch (err) {
			error = 'error: '+errmsg(err)
		}
		const l = []
		if (txt) {
			l.push(dom.div(dom._class('literal'), txt))
		}
		if (error) {
			l.push(box(red, error))
		}
		dom._kids(recordBox, l)
	})()
}

const tlsrptReports = async () => {
	const end = new Date()
	const start = new Date(new Date().getTime() - 30*24*3600*1000)
	const summaries = await client.TLSRPTSummaries(start, end, '')

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('TLSRPT', '#tlsrpt'),
			'Reports'
		),
		dom.p('TLSRPT (TLS reporting) is a mechanism to request feedback from other mail servers about TLS connections to your mail server. If is typically used along with MTA-STS and/or DANE to enforce that SMTP connections are protected with TLS. Mail servers implementing TLSRPT will typically send a daily report with both successful and failed connection counts, including details about failures.'),
		renderTLSRPTSummaries(summaries || [])
	)
}

const renderTLSRPTSummaries = (summaries: api.TLSRPTSummary[]) => {
	return [
		dom.p('Below a summary of TLS reports for the past 30 days.'),
		summaries.length === 0 ? dom.div(box(yellow, 'No domains with TLS reports.')) :
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('Policy domain', attr.title('Policy domain the report is about. The recipient domain for MTA-STS, the TLSA base domain for DANE.')),
					dom.th('Successes', attr.title('Number of successful SMTP STARTTLS sessions.')),
					dom.th('Failures', attr.title('Number of failed SMTP STARTTLS sessions.')),
					dom.th('Failure details', attr.title('Details about connection failures.')),
				)
			),
			dom.tbody(
				summaries.map(r =>
					dom.tr(
						dom.td(dom.a(attr.href('#tlsrpt/reports/' + domainName(r.PolicyDomain)), attr.title('See report details.'), domainName(r.PolicyDomain))),
						dom.td(style({textAlign: 'right'}), '' + r.Success),
						dom.td(style({textAlign: 'right'}), '' + r.Failure),
						dom.td(!r.ResultTypeCounts ? [] : Object.entries(r.ResultTypeCounts).map(kv => kv[0] + ': ' + kv[1]).join('; ')),
					)
				),
			),
		)
	]
}

const domainTLSRPT = async (d: string) => {
	const end = new Date()
	const start = new Date(new Date().getTime() - 30*24*3600*1000)
	const [records, dnsdomain] = await Promise.all([
		client.TLSReports(start, end, d),
		client.ParseDomain(d),
	])

	const policyType = (policy: api.ResultPolicy) => {
		let s: string = policy.Type
		if (s === 'sts') {
			const mode = (policy.String || []).find(s => s.startsWith('mode:'))
			if (mode) {
				s += ': '+mode.replace('mode:', '').trim()
			}
		}
		return s
	}

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('TLSRPT', '#tlsrpt'),
			crumblink('Reports', '#tlsrpt/reports'),
			'Domain '+domainString(dnsdomain),
		),
		dom.p('TLSRPT (TLS reporting) is a mechanism to request feedback from other mail servers about TLS connections to your mail server. If is typically used along with MTA-STS and/or DANE to enforce that SMTP connections are protected with TLS. Mail servers implementing TLSRPT will typically send a daily report with both successful and failed connection counts, including details about failures.'),
		dom.p('Below the TLS reports for the past 30 days.'),
		(records || []).length === 0 ? dom.div('No TLS reports for domain.') :
		dom.table(dom._class('hover'),
			dom.thead(
				dom.tr(
					dom.th('Report', attr.colspan('3')),
					dom.th('Policy', attr.colspan('3')),
					dom.th('Failure Details', attr.colspan('8')),
				),
				dom.tr(
					dom.th('ID'),
					dom.th('From', attr.title('SMTP mail from from which we received the report.')),
					dom.th('Period (UTC)', attr.title('Period this reporting period is about. Mail servers are recommended to stick to whole UTC days.')),

					dom.th('Policy', attr.title('The policy applied, typically STSv1.')),
					dom.th('Successes', attr.title('Total number of successful TLS connections for policy.')),
					dom.th('Failures', attr.title('Total number of failed TLS connections for policy.')),

					dom.th('Result Type', attr.title('Type of failure.')),
					dom.th('Sending MTA', attr.title('IP of sending MTA.')),
					dom.th('Receiving MX Host'),
					dom.th('Receiving MX HELO'),
					dom.th('Receiving IP'),
					dom.th('Count', attr.title('Number of TLS connections that failed with these details.')),
					dom.th('More', attr.title('Optional additional information about the failure.')),
					dom.th('Code', attr.title('Optional API error code relating to the failure.')),
				),
			),
			dom.tbody(
				(records || []).map(record => {
					const r = record.Report
					let nrows = 0
					;(r.Policies || []).forEach(pr => nrows += (pr.FailureDetails || []).length || 1)
					const reportRowSpan = attr.rowspan(''+nrows)
					const valignTop = style({verticalAlign: 'top'})
					const alignRight = style({textAlign: 'right'})
					return (r.Policies || []).map((result, index) => {
						const rows: HTMLElement[] = []
						const details = result.FailureDetails || []
						const resultRowSpan = attr.rowspan(''+(details.length || 1))
						const addRow = (d: api.FailureDetails | undefined, di: number) => {
							const row = dom.tr(
								index > 0 || rows.length > 0 ? [] : [
									dom.td(reportRowSpan, valignTop, dom.a(''+record.ID, attr.href('#tlsrpt/reports/' + record.Domain + '/' + record.ID))),
									dom.td(reportRowSpan, valignTop, r.OrganizationName || r.ContactInfo || record.MailFrom || '', attr.title('Organization: ' +r.OrganizationName + '; \nContact info: ' + r.ContactInfo + '; \nReport ID: ' + r.ReportID + '; \nMail from: ' + record.MailFrom)),
									dom.td(reportRowSpan, valignTop, period(r.DateRange.Start, r.DateRange.End)),
								],
								di > 0 ? [] : [
									dom.td(resultRowSpan, valignTop, policyType(result.Policy), attr.title((result.Policy.String || []).join('\n'))),
									dom.td(resultRowSpan, valignTop, alignRight, '' + result.Summary.TotalSuccessfulSessionCount),
									dom.td(resultRowSpan, valignTop, alignRight, '' + result.Summary.TotalFailureSessionCount),
								],
								!d ? dom.td(attr.colspan('8')) : [
									dom.td(d.ResultType),
									dom.td(d.SendingMTAIP),
									dom.td(d.ReceivingMXHostname),
									dom.td(d.ReceivingMXHelo),
									dom.td(d.ReceivingIP),
									dom.td(alignRight, '' + d.FailedSessionCount),
									dom.td(d.AdditionalInformation),
									dom.td(d.FailureReasonCode),

								],
							)
							rows.push(row)
						}
						let di = 0
						for (const d of details) {
							addRow(d, di)
							di++
						}
						if (details.length === 0) {
							addRow(undefined, 0)
						}
						return rows
					})
				})
			),
		)
	)
}

const domainTLSRPTID = async (d: string, reportID: number) => {
	const [report, dnsdomain] = await Promise.all([
		client.TLSReportID(d, reportID),
		client.ParseDomain(d),
	])

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			crumblink('TLSRPT', '#tlsrpt'),
			crumblink('Reports', '#tlsrpt/reports'),
			crumblink('Domain '+domainString(dnsdomain), '#tlsrpt/reports/' + d + ''),
			'Report ' + reportID
		),
		dom.p('Below is the raw report as received from the remote mail server.'),
		dom.div(dom._class('literal'), JSON.stringify(report, null, '\t')),
	)
}

const mtasts = async () => {
	const policies = await client.MTASTSPolicies()

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'MTA-STS policies',
		),
		dom.p("MTA-STS is a mechanism allowing email domains to publish a policy for using SMTP STARTTLS and TLS verification. See ", link('https://www.rfc-editor.org/rfc/rfc8461.html', 'RFC 8461'), '.'),
		dom.p("The SMTP protocol is unencrypted by default, though the SMTP STARTTLS command is typically used to enable TLS on a connection. However, MTA's using STARTTLS typically do not validate the TLS certificate. An MTA-STS policy can specify that validation of host name, non-expiration and webpki trust is required."),
		makeMTASTSTable(policies || []),
	)
}

const formatMTASTSMX = (mx: api.STSMX[]) => {
	return mx.map(e => {
		return (e.Wildcard ? '*.' : '') + e.Domain.ASCII
	}).join(', ')
}

const makeMTASTSTable = (items: api.PolicyRecord[]) => {
	if (items.length === 0) {
		return dom.div('No data')
	}
	// Elements: Field name in JSON, column name override, title for column name.
	const keys = [
		["LastUse", "", "Last time this policy was used."],
		["Domain", "Domain", "Domain this policy was retrieved from and this policy applies to."],
		["Backoff", "", "If true, a DNS record for MTA-STS exists, but a policy could not be fetched. This indicates a failure with MTA-STS."],
		["RecordID", "", "Unique ID for this policy. Each time a domain changes its policy, it must also change the record ID that is published in DNS to propagate the change."],
		["Version", "", "For valid MTA-STS policies, this must be 'STSv1'."],
		["Mode", "", "'enforce': TLS must be used and certificates must be validated; 'none': TLS and certificate validation is not required, typically only useful for removing once-used MTA-STS; 'testing': TLS should be used and certificated should be validated, but fallback to unverified TLS or plain text is allowed, but such cases must be reported"],
		["MX", "", "The MX hosts that are configured to do TLS. If TLS and validation is required, but an MX host is not on this list, delivery will not be attempted to that host."],
		["MaxAgeSeconds", "", "How long a policy can be cached and reused after it was fetched. Typically in the order of weeks."],
		["Extensions", "", "Free-form extensions in the MTA-STS policy."],
		["ValidEnd", "", "Until when this cached policy is valid, based on time the policy was fetched and the policy max age. Non-failure policies are automatically refreshed before they become invalid."],
		["LastUpdate", "", "Last time this policy was updated."],
		["Inserted", "", "Time when the policy was first inserted."],
	]
	const nowSecs = new Date().getTime()/1000
	return dom.table(dom._class('hover'),
		dom.thead(
			dom.tr(keys.map(kt => dom.th(dom.span(attr.title(kt[2]), kt[1] || kt[0])))),
		),
		dom.tbody(
			items.map(e =>
				dom.tr(
					[
						age(e.LastUse, false, nowSecs),
						e.Domain,
						e.Backoff,
						e.RecordID,
						e.Version,
						e.Mode,
						formatMTASTSMX(e.MX || []),
						e.MaxAgeSeconds,
						e.Extensions,
						age(e.ValidEnd, true, nowSecs),
						age(e.LastUpdate, false, nowSecs),
						age(e.Inserted, false, nowSecs),
					].map(v => dom.td(v === null ? [] : (v instanceof HTMLElement ? v : ''+v))),
				)
			),
		),
	)
}

const dnsbl = async () => {
	const ipZoneResults = await client.DNSBLStatus()

	const url = (ip: string) => {
		return 'https://multirbl.valli.org/lookup/' + encodeURIComponent(ip) + '.html'
	}

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'DNS blocklist status for IPs',
		),
		dom.p('Follow the external links to a third party DNSBL checker to see if the IP is on one of the many blocklist.'),
		dom.ul(
			Object.entries(ipZoneResults).sort().map(ipZones => {
				const [ip, zoneResults] = ipZones
				return dom.li(
					link(url(ip), ip),
					!ipZones.length ? [] : dom.ul(
						Object.entries(zoneResults).sort().map(zoneResult =>
							dom.li(
								zoneResult[0] + ': ',
								zoneResult[1] === 'pass' ? 'pass' : box(red, zoneResult[1]),
							),
						),
					),
				)
			})
		),
		!Object.entries(ipZoneResults).length ? box(red, 'No IPs found.') : [],
	)
}

const queueList = async () => {
	const [msgs, transports] = await Promise.all([
		client.QueueList(),
		client.Transports(),
	])

	const nowSecs = new Date().getTime()/1000

	dom._kids(page,
		crumbs(
			crumblink('Mox Admin', '#'),
			'Queue',
		),
		(msgs || []).length === 0 ? 'Currently no messages in the queue.' : [
			dom.p('The messages below are currently in the queue.'),
			// todo: sorting by address/timestamps/attempts. perhaps filtering.
			dom.table(dom._class('hover'),
				dom.thead(
					dom.tr(
						dom.th('ID'),
						dom.th('Submitted'),
						dom.th('From'),
						dom.th('To'),
						dom.th('Size'),
						dom.th('Attempts'),
						dom.th('Next attempt'),
						dom.th('Last attempt'),
						dom.th('Last error'),
						dom.th('Require TLS'),
						dom.th('Transport/Retry'),
						dom.th('Remove'),
					),
				),
				dom.tbody(
					(msgs || []).map(m => {
						let requiretlsFieldset: HTMLFieldSetElement
						let requiretls: HTMLSelectElement
						let transport: HTMLSelectElement
						return dom.tr(
							dom.td(''+m.ID),
							dom.td(age(new Date(m.Queued), false, nowSecs)),
							dom.td(m.SenderLocalpart+"@"+ipdomainString(m.SenderDomain)), // todo: escaping of localpart
							dom.td(m.RecipientLocalpart+"@"+ipdomainString(m.RecipientDomain)), // todo: escaping of localpart
							dom.td(formatSize(m.Size)),
							dom.td(''+m.Attempts),
							dom.td(age(new Date(m.NextAttempt), true, nowSecs)),
							dom.td(m.LastAttempt ? age(new Date(m.LastAttempt), false, nowSecs) : '-'),
							dom.td(m.LastError || '-'),
							dom.td(
								dom.form(
									requiretlsFieldset=dom.fieldset(
										requiretls=dom.select(
											attr.title('How to use TLS for message delivery over SMTP:\n\nDefault: Delivery attempts follow the policies published by the recipient domain: Verification with MTA-STS and/or DANE, or optional opportunistic unverified STARTTLS if the domain does not specify a policy.\n\nWith RequireTLS: For sensitive messages, you may want to require verified TLS. The recipient destination domain SMTP server must support the REQUIRETLS SMTP extension for delivery to succeed. It is automatically chosen when the destination domain mail servers of all recipients are known to support it.\n\nFallback to insecure: If delivery fails due to MTA-STS and/or DANE policies specified by the recipient domain, and the content is not sensitive, you may choose to ignore the recipient domain TLS policies so delivery can succeed.'),
											dom.option('Default', attr.value('')),
											dom.option('With RequireTLS', attr.value('yes'), m.RequireTLS === true ? attr.selected('') : []),
											dom.option('Fallback to insecure', attr.value('no'), m.RequireTLS === false ? attr.selected('') : []),
										),
										' ',
										dom.submitbutton('Save'),
									),
									async function submit(e: SubmitEvent) {
										e.preventDefault()
										try {
											requiretlsFieldset.disabled = true
											await client.QueueSaveRequireTLS(m.ID, requiretls.value === '' ? null : requiretls.value === 'yes')
										} catch (err) {
											console.log({err})
											window.alert('Error: ' + errmsg(err))
											return
										} finally {
											requiretlsFieldset.disabled = false
										}
									}
								),
							),
							dom.td(
								dom.form(
									transport=dom.select(
										attr.title('Transport to use for delivery attempts. The default is direct delivery, connecting to the MX hosts of the domain.'),
										dom.option('(default)', attr.value('')),
										Object.keys(transports || []).sort().map(t => dom.option(t, m.Transport === t ? attr.checked('') : [])),
									),
									' ',
									dom.submitbutton('Retry now'),
									async function submit(e: SubmitEvent) {
										e.preventDefault()
										const target = e.target! as HTMLButtonElement
										try {
											target.disabled = true
											await client.QueueKick(m.ID, transport.value)
										} catch (err) {
											console.log({err})
											window.alert('Error: ' + errmsg(err))
											return
										} finally {
											target.disabled = false
										}
										window.location.reload() // todo: only refresh the list
									}
								),
							),
							dom.td(
								dom.clickbutton('Remove', async function click(e: MouseEvent) {
									e.preventDefault()
									if (!window.confirm('Are you sure you want to remove this message? It will be removed completely.')) {
										return
									}
									const target = e.target! as HTMLButtonElement
									try {
										target.disabled = true
										await client.QueueDrop(m.ID)
									} catch (err) {
										console.log({err})
										window.alert('Error: ' + errmsg(err))
										return
									} finally {
										target.disabled = false
									}
									window.location.reload() // todo: only refresh the list
								}),
							),
						)
					})
				),
			),
		],
	)
}

const webserver = async () => {
	let conf = await client.WebserverConfig()

	// We disable this while saving the form.
	let fieldset: HTMLFieldSetElement

	type RedirectRow = {
		root: HTMLElement
		from: HTMLInputElement
		to: HTMLInputElement
		get: () => [string, string]
	}

	// Keep track of redirects. Rows are objects that hold both the DOM and allows
	// retrieving the visible (modified) data to construct a config for saving.
	let redirectRows: RedirectRow[] = []
	let redirectsTbody: HTMLElement
	let noredirect: HTMLElement

	// Make a new redirect rows, adding it to the list. The caller typically uses this
	// while building the DOM, the element is added because this object has it as
	// "root" field.
	const redirectRow = (t: [api.Domain, api.Domain]) => {
		let row: RedirectRow
		let from: HTMLInputElement
		let to: HTMLInputElement

		const root = dom.tr(
			dom.td(
				from=dom.input(attr.required(''), attr.value(domainName(t[0]))),
			),
			dom.td(
				to=dom.input(attr.required(''), attr.value(domainName(t[1]))),
			),
			dom.td(
				dom.clickbutton('Remove', function click() {
					redirectRows = redirectRows.filter(r => r !== row)
					row.root.remove()
					noredirect.style.display = redirectRows.length ? 'none' : ''
				}),
			),
		)
		// "get" is the common function to retrieve the data from an object with a root field as DOM element.
		const get = (): [string, string] => [from.value, to.value]
		row = {root: root, from: from, to: to, get: get}
		redirectRows.push(row)
		return row
	}

	type HeadersView = {
		root: HTMLElement

		add: HTMLButtonElement
		get: () => { [k: string]: string }
	}

	// Reusable component for managing headers. Just a table with a header key and
	// value. We can remove existing rows, and add new rows, and edit existing.
	const makeHeaders = (h: { [key: string]: string }) => {
		let view: HeadersView
		let rows: HeaderRow[] = []
		let tbody: HTMLElement
		let norow: HTMLElement

		type HeaderRow = {
			root: HTMLElement
			key: HTMLInputElement
			value: HTMLInputElement
			get: () => [string, string]
		}

		const headerRow = (k: string, v: string) => {
			let row: HeaderRow
			let key: HTMLInputElement
			let value: HTMLInputElement

			const root = dom.tr(
				dom.td(
					key=dom.input(attr.required(''), attr.value(k)),
				),
				dom.td(
					value=dom.input(attr.required(''), attr.value(v)),
				),
				dom.td(
					dom.clickbutton('Remove', function click() {
						rows = rows.filter(x => x !== row)
						row.root.remove()
						norow.style.display = rows.length ? 'none' : ''
					})
				),
			)
			const get = (): [string, string] => [row.key.value, row.value.value]
			row = {root: root, key: key, value: value, get: get}
			rows.push(row)
			return row
		}
		const add = dom.clickbutton('Add', function click() {
			const row = headerRow('', '')
			tbody.appendChild(row.root)
			norow.style.display = rows.length ? 'none' : ''
		})
		const root = dom.table(
			tbody=dom.tbody(
				Object.entries(h).sort().map(t => headerRow(t[0], t[1])),
				norow=dom.tr(
					style({display: rows.length ? 'none' : ''}),
					dom.td(attr.colspan('3'), 'None added.'),
				)
			),
		)
		const get = (): { [k: string]: string } => Object.fromEntries(rows.map(row => row.get()))
		view = {root: root, add: add, get: get}
		return view
	}


	// todo: make a mechanism to get the ../config/config.go sconf-doc struct tags
	// here. So we can use them for the titles, as documentation. Instead of current
	// approach of copy/pasting those texts, inevitably will get out of date.

	// todo: perhaps lay these out in the same way as in the config file? will help admins mentally map between the two. will take a bit more vertical screen space, but current approach looks messy/garbled. we could use that mechanism for more parts of the configuration file. we can even show the same sconf-doc struct tags. the html admin page will then just be a glorified guided text editor!

	type HandlerRow = {
		root: HTMLElement
		moveButtons: HTMLElement
		get: () => api.WebHandler
	}

	// Similar to redirects, but for web handlers.
	let handlerRows: HandlerRow[] = []
	let handlersTbody: HTMLElement
	let nohandler: HTMLElement

	type WebStaticView = {
		root: HTMLElement
		get: () => api.WebStatic
	}
	type WebRedirectView = {
		root: HTMLElement
		get: () => api.WebRedirect
	}
	type WebForwardView = {
		root: HTMLElement
		get: () => api.WebForward
	}

	// Make a handler row. This is more complicated, since it can be one of the three
	// types (static, redirect, forward), and can change between those types.
	const handlerRow = (wh: api.WebHandler) => {
		let row: HandlerRow // Shared between the handler types.
		let handlerType: string
		let staticView: WebStaticView | null = null
		let redirectView: WebRedirectView | null = null
		let forwardView: WebForwardView | null = null

		let moveButtons: HTMLElement

		const makeWebStatic = (ws: api.WebStatic) => {
			let view: WebStaticView

			let stripPrefix: HTMLInputElement
			let rootPath: HTMLInputElement
			let listFiles: HTMLInputElement
			let continueNotFound: HTMLInputElement
			let responseHeaders: HeadersView = makeHeaders(ws.ResponseHeaders || {})

			const get = (): api.WebStatic => {
				return {
					StripPrefix: stripPrefix.value,
					Root: rootPath.value,
					ListFiles: listFiles.checked,
					ContinueNotFound: continueNotFound.checked,
					ResponseHeaders: responseHeaders.get(),
				}
			}
			const root = dom.table(
				dom.tr(
					dom.td('Type'),
					dom.td(
						'StripPrefix',
						attr.title('Path to strip from the request URL before evaluating to a local path. If the requested URL path does not start with this prefix and ContinueNotFound it is considered non-matching and next WebHandlers are tried. If ContinueNotFound is not set, a file not found (404) is returned in that case.'),
					),
					dom.td(
						'Root',
						attr.title('Directory to serve files from for this handler. Keep in mind that relative paths are relative to the working directory of mox.'),
					),
					dom.td(
						'ListFiles',
						attr.title('If set, and a directory is requested, and no index.html is present that can be served, a file listing is returned. Results in 403 if ListFiles is not set. If a directory is requested and the URL does not end with a slash, the response is a redirect to the path with trailing slash.'),
					),
					dom.td(
						'ContinueNotFound',
						attr.title("If a requested URL does not exist, don't return a file not found (404) response, but consider this handler non-matching and continue attempts to serve with later WebHandlers, which may be a reverse proxy generating dynamic content, possibly even writing a static file for a next request to serve statically. If ContinueNotFound is set, HTTP requests other than GET and HEAD do not match. This mechanism can be used to implement the equivalent of 'try_files' in other webservers."),
					),
					dom.td(
						dom.span(
							'Response headers',
							attr.title('Headers to add to the response. Useful for cache-control, content-type, etc. By default, Content-Type headers are automatically added for recognized file types, unless added explicitly through this setting. For directory listings, a content-type header is skipped.'),
						),
						' ',
						responseHeaders.add,
					),
				),
				dom.tr(
					dom.td(
						dom.select(
							attr.required(''),
							dom.option('Static', attr.selected('')),
							dom.option('Redirect'),
							dom.option('Forward'),
							function change(e: MouseEvent) {
								makeType((e.target! as HTMLSelectElement).value)
							},
						),
					),
					dom.td(
						stripPrefix=dom.input(attr.value(ws.StripPrefix || '')),
					),
					dom.td(
						rootPath=dom.input(attr.required(''), attr.placeholder('web/...'), attr.value(ws.Root || '')),
					),
					dom.td(
						listFiles=dom.input(attr.type('checkbox'), ws.ListFiles ? attr.checked('') : []),
					),
					dom.td(
						continueNotFound=dom.input(attr.type('checkbox'), ws.ContinueNotFound ? attr.checked('') : []),
					),
					dom.td(
						responseHeaders,
					),
				)
			)
			view = {root: root, get: get}
			return view
		}

		const makeWebRedirect = (wr: api.WebRedirect) => {
			let view: WebRedirectView

			let baseURL: HTMLInputElement
			let origPathRegexp: HTMLInputElement
			let replacePath: HTMLInputElement
			let statusCode: HTMLInputElement

			const get = (): api.WebRedirect => {
				return {
					BaseURL: baseURL.value,
					OrigPathRegexp: origPathRegexp.value,
					ReplacePath: replacePath.value,
					StatusCode: statusCode.value ? parseInt(statusCode.value) : 0,
				}
			}
			const root = dom.table(
				dom.tr(
					dom.td('Type'),
					dom.td(
						'BaseURL',
						attr.title('Base URL to redirect to. The path must be empty and will be replaced, either by the request URL path, or by OrigPathRegexp/ReplacePath. Scheme, host, port and fragment stay intact, and query strings are combined. If empty, the response redirects to a different path through OrigPathRegexp and ReplacePath, which must then be set. Use a URL without scheme to redirect without changing the protocol, e.g. //newdomain/. If a redirect would send a request to a URL with the same scheme, host and path, the WebRedirect does not match so a next WebHandler can be tried. This can be used to redirect all plain http traffic to https.'),
					),
					dom.td(
						'OrigPathRegexp',
						attr.title('Regular expression for matching path. If set and path does not match, a 404 is returned. The HTTP path used for matching always starts with a slash.'),
					),
					dom.td(
						'ReplacePath',
						attr.title("Replacement path for destination URL based on OrigPathRegexp. Implemented with Go's Regexp.ReplaceAllString: $1 is replaced with the text of the first submatch, etc. If both OrigPathRegexp and ReplacePath are empty, BaseURL must be set and all paths are redirected unaltered."),
					),
					dom.td(
						'StatusCode',
						attr.title('Status code to use in redirect, e.g. 307. By default, a permanent redirect (308) is returned.'),
					),
				),
				dom.tr(
					dom.td(
						dom.select(
							attr.required(''),
							dom.option('Static'),
							dom.option('Redirect', attr.selected('')),
							dom.option('Forward'),
							function change(e: MouseEvent) {
								makeType((e.target! as HTMLSelectElement).value)
							},
						),
					),
					dom.td(
						baseURL=dom.input(attr.placeholder('empty or https://target/path?q=1#frag or //target/...'), attr.value(wr.BaseURL || '')),
					),
					dom.td(
						origPathRegexp=dom.input(attr.placeholder('^/old/(.*)'), attr.value(wr.OrigPathRegexp || '')),
					),
					dom.td(
						replacePath=dom.input(attr.placeholder('/new/$1'), attr.value(wr.ReplacePath || '')),
					),
					dom.td(
						statusCode=dom.input(style({width: '4em'}), attr.type('number'), attr.value(wr.StatusCode ? ''+wr.StatusCode : ''), attr.min('300'), attr.max('399')),
					),
				),
			)
			view = {root: root, get: get}
			return view
		}

		const makeWebForward = (wf: api.WebForward) => {
			let view: WebForwardView

			let stripPath: HTMLInputElement
			let url: HTMLInputElement
			let responseHeaders: HeadersView = makeHeaders(wf.ResponseHeaders || {})

			const get = (): api.WebForward => {
				return {
					StripPath: stripPath.checked,
					URL: url.value,
					ResponseHeaders: responseHeaders.get(),
				}
			}
			const root = dom.table(
				dom.tr(
					dom.td('Type'),
					dom.td(
						'StripPath',
						attr.title('Strip the matching WebHandler path from the WebHandler before forwarding the request.'),
					),
					dom.td(
						'URL',
						attr.title("URL to forward HTTP requests to, e.g. http://127.0.0.1:8123/base. If StripPath is false the full request path is added to the URL. Host headers are sent unmodified. New X-Forwarded-{For,Host,Proto} headers are set. Any query string in the URL is ignored. Requests are made using Go's net/http.DefaultTransport that takes environment variables HTTP_PROXY and HTTPS_PROXY into account. Websocket connections are forwarded and data is copied between client and backend without looking at the framing. The websocket 'version' and 'key'/'accept' headers are verified during the handshake, but other websocket headers, including 'origin', 'protocol' and 'extensions' headers, are not inspected and the backend is responsible for verifying/interpreting them."),
					),
					dom.td(
						dom.span(
							'Response headers',
							attr.title('Headers to add to the response. Useful for adding security- and cache-related headers.'),
						),
						' ',
						responseHeaders.add,
					),
				),
				dom.tr(
					dom.td(
						dom.select(
							attr.required(''),
							dom.option('Static', ),
							dom.option('Redirect'),
							dom.option('Forward', attr.selected('')),
							function change(e: MouseEvent) {
								makeType((e.target! as HTMLSelectElement).value)
							},
						),
					),
					dom.td(
						stripPath=dom.input(attr.type('checkbox'), wf.StripPath || wf.StripPath === undefined ? attr.checked('') : []),
					),
					dom.td(
						url=dom.input(attr.required(''), attr.placeholder('http://127.0.0.1:8888'), attr.value(wf.URL || '')),
					),
					dom.td(
						responseHeaders,
					),
				),
			)
			view = {root: root, get: get}
			return view
		}

		let logName: HTMLInputElement
		let domain: HTMLInputElement
		let pathRegexp: HTMLInputElement
		let toHTTPS: HTMLInputElement
		let compress: HTMLInputElement

		let details: HTMLElement

		const detailsRoot = (root: HTMLElement) => {
			details.replaceWith(root)
			details = root
		}

		// Transform the input fields to match the type of WebHandler.
		const makeType = (s: string) => {
			if (s === 'Static') {
				staticView = makeWebStatic(wh.WebStatic || {
					StripPrefix: '',
					Root: '',
					ListFiles: false,
					ContinueNotFound: false,
					ResponseHeaders: {},
				})
				detailsRoot(staticView.root)
			} else if (s === 'Redirect') {
				redirectView = makeWebRedirect(wh.WebRedirect || {
					BaseURL: '',
					OrigPathRegexp: '',
					ReplacePath: '',
					StatusCode: 0,
				})
				detailsRoot(redirectView.root)
			} else if (s === 'Forward') {
				forwardView = makeWebForward(wh.WebForward || {
					StripPath: false,
					URL: '',
					ResponseHeaders: {},
				})
				detailsRoot(forwardView.root)
			} else {
				throw new Error('unknown handler type')
			}
			handlerType = s
		}

		// Remove row from oindex, insert it in nindex. Both in handlerRows and in the DOM.
		const moveHandler = (row: HandlerRow, oindex: number, nindex: number) => {
			row.root.remove()
			handlersTbody.insertBefore(row.root, handlersTbody.children[nindex])
			handlerRows.splice(oindex, 1)
			handlerRows.splice(nindex, 0, row)
		}

		// Row that starts starts with two tables: one for the fields all WebHandlers have
		// (in common). And one for the details, i.e. WebStatic, WebRedirect, WebForward.
		const root = dom.tr(
			dom.td(
				dom.table(
					dom.tr(
						dom.td('LogName', attr.title('Name used during logging for requests matching this handler. If empty, the index of the handler in the list is used.')),
						dom.td('Domain', attr.title('Request must be for this domain to match this handler.')),
						dom.td('Path Regexp', attr.title('Request must match this path regular expression to match this handler. Must start with with a ^.')),
						dom.td('To HTTPS', attr.title('Redirect plain HTTP (non-TLS) requests to HTTPS.')),
						dom.td('Compress', attr.title('Transparently compress responses (currently with gzip) if the client supports it, the status is 200 OK, no Content-Encoding is set on the response yet and the Content-Type of the response hints that the data is compressible (text/..., specific application/... and .../...+json and .../...+xml). For static files only, a cache with compressed files is kept.')),
					),
					dom.tr(
						dom.td(
							logName=dom.input(attr.value(wh.LogName || '')),
						),
						dom.td(
							domain=dom.input(attr.required(''), attr.placeholder('example.org'), attr.value(domainName(wh.DNSDomain))),
						),
						dom.td(
							pathRegexp=dom.input(attr.required(''), attr.placeholder('^/'), attr.value(wh.PathRegexp || '')),
						),
						dom.td(
							toHTTPS=dom.input(attr.type('checkbox'), attr.title('Redirect plain HTTP (non-TLS) requests to HTTPS'), !wh.DontRedirectPlainHTTP ? attr.checked('') : []),
						),
						dom.td(
							compress=dom.input(attr.type('checkbox'), attr.title('Transparently compress responses.'), wh.Compress ? attr.checked('') : []),
						),
					),
				),
				// Replaced with a call to makeType, below (and later when switching types).
				details=dom.table(),
			),
			dom.td(
				dom.td(
					dom.clickbutton('Remove', function click() {
						handlerRows = handlerRows.filter(r => r !== row)
						row.root.remove()
						nohandler.style.display = handlerRows.length ? 'none' : ''
					}),
					' ',
					// We show/hide the buttons to move when clicking the Move button.
					moveButtons=dom.span(
						style({display: 'none'}),
						dom.clickbutton('↑↑', attr.title('Move to top.'), function click() {
							const index = handlerRows.findIndex(r => r === row)
							if (index > 0) {
								moveHandler(row, index, 0)
							}
						}),
						' ',
						dom.clickbutton('↑', attr.title('Move one up.'), function click() {
							const index = handlerRows.findIndex(r => r === row)
							if (index > 0) {
								moveHandler(row, index, index-1)
							}
						}),
						' ',
						dom.clickbutton('↓', attr.title('Move one down.'), function click() {
							const index = handlerRows.findIndex(r => r === row)
							if (index+1 < handlerRows.length) {
								moveHandler(row, index, index+1)
							}
						}),
						' ',
						dom.clickbutton('↓↓', attr.title('Move to bottom.'), function click() {
							const index = handlerRows.findIndex(r => r === row)
							if (index+1 < handlerRows.length) {
								moveHandler(row, index, handlerRows.length-1)
							}
						}),
					),
				),
			),
		)

		// Final "get" that returns a WebHandler that reflects the UI.
		const get = (): api.WebHandler => {
			const wh: api.WebHandler = {
				LogName: logName.value,
				Domain: domain.value,
				PathRegexp: pathRegexp.value,
				DontRedirectPlainHTTP: !toHTTPS.checked,
				Compress: compress.checked,
				Name: '',
				DNSDomain: {ASCII: '', Unicode: ''},
			}
			if (handlerType === 'Static' && staticView != null) {
				wh.WebStatic = staticView.get()
			} else if (handlerType === 'Redirect' && redirectView !== null) {
				wh.WebRedirect = redirectView.get()
			} else if (handlerType === 'Forward' && forwardView !== null) {
				wh.WebForward = forwardView.get()
			} else {
				throw new Error('unknown WebHandler type')
			}
			return wh
		}

		// Initialize one of the Web* types.
		if (wh.WebStatic) {
			handlerType = 'Static'
		} else if (wh.WebRedirect) {
			handlerType = 'Redirect'
		} else if (wh.WebForward) {
			handlerType = 'Forward'
		} else {
			throw new Error('unknown WebHandler type')
		}
		makeType(handlerType)

		row = {root: root, moveButtons: moveButtons, get: get}
		handlerRows.push(row)
		return row
	}

	// Return webserver config to store.
	const gatherConf = () => {
		return {
			WebDomainRedirects: redirectRows.map(row => row.get()),
			WebHandlers: handlerRows.map(row => row.get()),
		}
	}

	// Add and move buttons, both above and below the table for quick access, hence a function.
	const handlerActions = () => {
		return [
			'Action ',
			dom.clickbutton('Add', function click() {
				// New WebHandler added as WebForward. Good chance this is what the user wants. And
				// it has the least fields. (;
				const nwh: api.WebHandler = {
					LogName: '',
					Domain: '',
					PathRegexp: '^/',
					DontRedirectPlainHTTP: false,
					Compress: false,
					WebForward: {
						StripPath: true,
						URL: '',
					},
					Name: '',
					DNSDomain: {ASCII: '', Unicode: ''},
				}
				const row = handlerRow(nwh)
				handlersTbody.appendChild(row.root)
				nohandler.style.display = handlerRows.length ? 'none' : ''
			}),
			' ',
			dom.clickbutton('Move', function click() {
				for(const row of handlerRows) {
					row.moveButtons.style.display = row.moveButtons.style.display === 'none' ? '' : 'none'
				}
			}),
		]
	}

	dom._kids(page,
		crumbs(
			crumblink('Beacon Admin', '#'),
			'Webserver config',
		),
		dom.form(
			fieldset=dom.fieldset(
				dom.h2('Domain redirects', attr.title('Corresponds with WebDomainRedirects in domains.conf')),
				dom.p('Incoming requests for these domains are redirected to the target domain, with HTTPS.'),
				dom.table(
					dom.thead(
						dom.tr(
							dom.th('From'),
							dom.th('To'),
							dom.th(
								'Action ',
								dom.clickbutton('Add', function click() {
									const row = redirectRow([{ASCII: '', Unicode: ''}, {ASCII: '', Unicode: ''}])
									redirectsTbody.appendChild(row.root)
									noredirect.style.display = redirectRows.length ? 'none' : ''
								}),
							),
						),
					),
					redirectsTbody=dom.tbody(
						(conf.WebDNSDomainRedirects || []).sort().map(t => redirectRow([t![0], t![1]])),
						noredirect=dom.tr(
							style({display: redirectRows.length ? 'none' : ''}),
							dom.td(attr.colspan('3'), 'No redirects.'),
						),
					),
				),
				dom.br(),
				dom.h2('Handlers', attr.title('Corresponds with WebHandlers in domains.conf')),
				dom.p('Each incoming request is check against these handlers, in order. The first matching handler serves the request. Don\'t forget to save after making a change.'),
				dom.table(dom._class('long'),
					dom.thead(
						dom.tr(
							dom.th(),
							dom.th(handlerActions()),
						),
					),
					handlersTbody=dom.tbody(
						(conf.WebHandlers || []).map(wh => handlerRow(wh)),
						nohandler=dom.tr(
							style({display: handlerRows.length ? 'none' : ''}),
							dom.td(attr.colspan('2'), 'No handlers.'),
						),
					),
					dom.tfoot(
						dom.tr(
							dom.th(),
							dom.th(handlerActions()),
						),
					),
				),
				dom.br(),
				dom.submitbutton('Save', attr.title('Save config. If the configuration has changed since this page was loaded, an error will be returned. After saving, the changes take effect immediately.')),
			),
			async function submit(e: SubmitEvent) {
				e.preventDefault()
				e.stopPropagation()

				fieldset.disabled = true
				try {
					const newConf = gatherConf()
					const savedConf = await client.WebserverConfigSave(conf, newConf)
					conf = savedConf
				} catch (err) {
					console.log({err})
					window.alert('Error: ' + errmsg(err))
				} finally {
					fieldset.disabled = false
				}
			}
		),
	)
}

const init = async () => {
	let curhash: string | undefined

	const hashChange = async () => {
		if (curhash === window.location.hash) {
			return
		}
		let h = decodeURIComponent(window.location.hash)
		if (h !== '' && h.substring(0, 1) == '#') {
			h = h.substring(1)
		}
		const t = h.split('/')
		page.classList.add('loading')
		try {
			if (h == '') {
				await index()
			} else if (h === 'config') {
				await config()
			} else if (h === 'loglevels') {
				await loglevels()
			} else if (h === 'accounts') {
				await accounts()
			} else if (t[0] === 'accounts' && t.length === 2) {
				await account(t[1])
			} else if (t[0] === 'domains' && t.length === 2) {
				await domain(t[1])
			} else if (t[0] === 'domains' && t.length === 3 && t[2] === 'dmarc') {
				await domainDMARC(t[1])
			} else if (t[0] === 'domains' && t.length === 4 && t[2] === 'dmarc' && parseInt(t[3])) {
				await domainDMARCReport(t[1], parseInt(t[3]))
			} else if (t[0] === 'domains' && t.length === 3 && t[2] === 'dnscheck') {
				await domainDNSCheck(t[1])
			} else if (t[0] === 'domains' && t.length === 3 && t[2] === 'dnsrecords') {
				await domainDNSRecords(t[1])
			} else if (h === 'queue') {
				await queueList()
			} else if (h === 'tlsrpt') {
				await tlsrptIndex()
			} else if (h === 'tlsrpt/reports') {
				await tlsrptReports()
			} else if (t[0] === 'tlsrpt' && t[1] === 'reports' && t.length === 3) {
				await domainTLSRPT(t[2])
			} else if (t[0] === 'tlsrpt' && t[1] === 'reports' && t.length === 4 && parseInt(t[3])) {
				await domainTLSRPTID(t[2], parseInt(t[3]))
			} else if (h === 'tlsrpt/results') {
				await tlsrptResults()
			} else if (t[0] == 'tlsrpt' && t[1] == 'results' && (t[2] === 'rcptdom' || t[2] == 'host') && t.length === 4) {
				await tlsrptResultsPolicyDomain(t[2] === 'rcptdom', t[3])
			} else if (h === 'dmarc') {
				await dmarcIndex()
			} else if (h === 'dmarc/reports') {
				await dmarcReports()
			} else if (h === 'dmarc/evaluations') {
				await dmarcEvaluations()
			} else if (t[0] == 'dmarc' && t[1] == 'evaluations' && t.length === 3) {
				await dmarcEvaluationsDomain(t[2])
			} else if (h === 'mtasts') {
				await mtasts()
			} else if (h === 'dnsbl') {
				await dnsbl()
			} else if (h === 'webserver') {
				await webserver()
			} else {
				dom._kids(page, 'page not found')
			}
		} catch (err) {
			console.log('error', err)
			window.alert('Error: ' + errmsg(err))
			curhash = window.location.hash
			return
		}
		curhash = window.location.hash
		page.classList.remove('loading')
	}
	window.addEventListener('hashchange', hashChange)
	hashChange()
}

window.addEventListener('load', init)
