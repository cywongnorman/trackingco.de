const React = require('react')
const h = require('react-hyperscript')
const withClickOutside = require('react-click-outside')
const TwitterPicker = require('react-color').TwitterPicker
const randomColor = require('randomcolor')
const n = require('format-number')({prefix: '$'})
const fecha = require('fecha')
const DocumentTitle = require('react-document-title')
const BodyStyle = require('body-style')

const log = require('./log')
const graphql = require('./graphql')
const mergeColours = require('./helpers').mergeColours
const coloursfragment = require('./helpers').coloursfragment
const title = require('./helpers').title
const onLoggedStateChange = require('./auth').onLoggedStateChange

const plans = [
  {name: 'Free', code: 0},
  {name: 'First', code: 1},
  {name: 'Second', code: 2},
  {name: 'Third', code: 3}
]

const prettydate = iso => fecha.format(new Date(iso), 'mediumDate')

const UserAccount = React.createClass({
  getInitialState () {
    return {
      me: null,
      newDomain: '',
      willPay: 12
    }
  },

  query () {
    graphql.query(`
query {
  me {
    colours { ...${coloursfragment} }
    domains
    email
    plan
    billingHistory { id, time, delta, due }
  }
}
    `)
    .then(r => this.setState(r))
    .catch(log.error)
  },

  componentDidMount () {
    onLoggedStateChange(isLogged => {
      if (isLogged) {
        this.query()
      }
    })
  },

  render () {
    if (!this.state.me) {
      return h('div')
    }

    let backgroundColor = mergeColours(this.state.me.colours).background

    return (
      h(DocumentTitle, {title: title('User account')}, [
        h(BodyStyle, {style: {backgroundColor}}, [
          h('.tile.is-ancestor.is-vertical', [
            h('.tile.is-12', [
              h('.tile', [
                h('.tile.is-parent', [
                  h('.tile.is-child', [
                    h('.card.account-domains', [
                      h('.card-header', [
                        h('.card-header-title', 'Registered domains')
                      ]),
                      h('.card-content', [
                        this.state.me.plan < 2
                        ? h('p', 'Please upgrade your account to gain access to this feature.')
                        : (
                          h('ul', this.state.me.domains.map(hostname =>
                            h('li', {key: hostname}, [
                              hostname + ' ',
                              h('a.delete', {onClick: e => { this.removeDomain(e, hostname) }})
                            ])
                          ).concat(
                            h('form', {key: '~', onSubmit: this.addDomain}, [
                              h('p.control.has-icon', [
                                h('input.input', {
                                  type: 'text',
                                  onChange: e => { this.setState({newDomain: e.target.value}) },
                                  value: this.state.newDomain,
                                  placeholder: 'Add a domain or subdomain'
                                }),
                                h('span.icon.is-small', [
                                  h('i.fa.fa-plus')
                                ])
                              ]),
                              h('p.control', [
                                h('button.button', 'Save')
                              ])
                            ])
                          ))
                        )
                      ])
                    ])
                  ])
                ])
              ]),
              h('.tile', [
                h('.tile.is-parent', [
                  h('.tile.is-child', [
                    h('.card.account-colours', [
                      h('.card-header', [
                        h('.card-header-title', 'Interface colours')
                      ]),
                      h('.card-content', [
                        this.state.me.plan < 1
                        ? h('p', 'Please upgrade your account to gain access to this feature.')
                        : h(Colours, {
                          ...this.state,
                          onColour: this.changeColour
                        })
                      ])
                    ])
                  ])
                ]),
                h('.tile.is-parent', [
                  h('.tile.is-child', [
                    h('.card.account-info', [
                      h('.card-header', [
                        h('.card-header-title', 'Account information')
                      ]),
                      h('.card-content', [
                        h('p', `email: ${this.state.me.email}`),
                        h('p', `plan: ${plans[this.state.me.plan].name}`)
                      ])
                    ])
                  ])
                ])
              ])
            ]),
            h('.tile.is-12', [
              h('.tile.is-4', [
                h('.tile.is-parent', [
                  h('.tile.is-child', [
                    h('.card.account-plan', [
                      h('.card-header', [
                        h('.card-header-title', 'Your plan')
                      ]),
                      h('.card-content', [
                        h('h2.title.is-2', `${plans[this.state.me.plan].name} Plan`),
                        this.state.me.plan >= (plans.length - 1)
                        ? h('p', 'Contact us if you want a bigger plan.')
                        : h('div', [
                          h('h3.title.is-3', 'Upgrade to:'),
                          h('ul', plans.slice(this.state.me.plan + 1).map(({name, code}) =>
                            h('li', [
                              h('button.button.is-large.is-warning', {
                                onClick: e => this.setPlan(code, e)
                              }, `${name} Plan`)
                            ])
                          ))
                        ]),
                        h('hr'),
                        this.state.me.plan === 0
                        ? 'You have no guarantees in the Free Plan, all your data can be erased at any time.'
                        : h('div', [
                          h('h6.title.is-6', 'Downgrade to:'),
                          h('ul', plans.slice(0, this.state.me.plan).map(({name, code}) =>
                            h('li', [
                              h('button.button.is-small.is-light', {
                                onClick: e => this.setPlan(code, e)
                              }, `${name} Plan`)
                            ])
                          ))
                        ])
                      ])
                    ])
                  ])
                ])
              ]),
              h('.tile.is-6', [
                h('.tile.is-parent', [
                  h('.tile.is-child', [
                    h('.card.account-billing-history', [
                      h('.card-header', [
                        h('.card-header-title', 'Billing History')
                      ]),
                      h('.card-content', [
                        h('h3.title.is-3', [
                          h('small.small', 'Balance: '),
                          n(this.state.me.billingHistory.reduce((acc, entry) => acc + entry.delta, 0), 2)
                        ]),
                        h('table.table', [
                          h('thead', [
                            h('tr', [
                              h('th', 'date'),
                              h('th', 'kind'),
                              h('th', 'value'),
                              h('th', 'valid until')
                            ])
                          ]),
                          h('tbody', this.state.me.billingHistory.map(entry =>
                            h('tr', {key: entry.id, id: entry.id}, [
                              h('td', prettydate(entry.time)),
                              h('td', entry.due ? 'charge' : 'payment'),
                              h('td', n(entry.delta)),
                              h('td', entry.due ? prettydate(entry.due) : '')
                            ])
                          ))
                        ]),
                        h('hr'),
                        h('.payment', [
                          h('p.control', [
                            h('a.button.is-static.is-disabled', '$')
                          ]),
                          h('p.control', [
                            h('input.input', {
                              onChange: e => this.setState({willPay: e.target.value}),
                              value: this.state.willPay
                            })
                          ]),
                          h('p.control', [
                            h('a.button.is-info', {
                              onClick: this.bitcoinPayRedirect
                            }, 'Make a Bitcoin payment')
                          ])
                        ])
                      ])
                    ])
                  ])
                ])
              ]),
              h('.tile.is-2')
            ])
          ])
        ])
      ])
    )
  },

  setPlan (code, e) {
    e.preventDefault()

    if (this.state.me.billingHistory.reduce((acc, e) => acc + e.delta, 0) <= 0) {
      log.info('Please make a payment to fund your account before upgrading!')
      return
    }

    graphql.mutate(`
($code: Float!) {
  setPlan(plan: $code) {
    ok, error
  }
}
    `, {code})
    .then(r => {
      if (!r.setPlan.ok) {
        log.error('failed to setPlan:', r.setPlan.error)
        return
      }
      this.setState(st => {
        st.me.plan = code
        return st
      })
    })
    .catch(log.error)
  },

  bitcoinPayRedirect (e) {
    e.preventDefault()

    log.info("We're going to redirect you to our Bitcoin payments provider.")

    graphql.mutate(`
($value: Float!) {
  makePayment(value: $value)
}
    `, {value: this.state.willPay})
    .then(r => {
      if (r.makePayment && r.makePayment.slice(0, 4) === 'http') {
        location.href = r.makePayment
      } else {
        throw new Error(r.makePayment)
      }
    })
    .catch(log.error)
  },

  changeColour (field, colour) {
    graphql.mutate(`
($colours: ColoursInput!) {
  setColours(colours: $colours) {
    ok, error
  }
}
    `, {colours: {...this.state.me.colours, ...{[field]: colour}}})
    .then(r => {
      if (!r.setColours.ok) {
        log.error('failed to setColours:', r.setColours.error)
        return
      }
      this.setState(st => {
        st.me.colours[field] = colour
        return st
      })
    })
    .catch(log.error)
  },

  addDomain (e) {
    e.preventDefault()
    let hostname = this.state.newDomain

    graphql.mutate(`
($hostname: String!) {
  addDomain(hostname: $hostname) {
    ok, error
  }
}
    `, {hostname})
    .then(r => {
      if (!r.addDomain.ok) {
        log.error('error adding domain: ', r.addDomain.error)
        return
      }
      log.info(hostname, 'added.')
      this.query()
    })
    .catch(log.error)
  },

  removeDomain (e, host) {
    e.preventDefault()

    graphql.mutate(`
($host: String!) {
  removeDomain(hostname: $host) {
    ok, error
  }
}
    `, {host})
    .then(r => {
      if (!r.removeDomain.ok) {
        log.error('error removing domain:', r.removeDomain.error)
        return
      }
      log.info(host, 'removed.')
      this.query()
    })
    .catch(log.error)
  }
})

const Colours = withClickOutside(React.createClass({
  getInitialState () {
    return {
      display: null
    }
  },

  render () {
    let colours = mergeColours(this.props.me.colours)

    return (
      h('.colours', [
        Object.keys(colours).map(field =>
          h('div', {key: field}, [
            h('a', {
              onClick: (e) => {
                e.preventDefault()
                this.setState({display: field})
              }
            }, [
              h('i.fa.fa-square', {
                style: {
                  color: colours[field],
                  fontSize: '50px'
                }
              })
            ]),
            field === this.state.display && h(TwitterPicker, {
              colors: randomColor({count: 10}),
              onChange: ({hex}) => this.props.onColour(field, hex)
            })
          ])
        )
      ])
    )
  },

  handleClickOutside () {
    this.setState({display: null})
  }
}))

module.exports = UserAccount
