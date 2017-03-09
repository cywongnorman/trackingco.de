const React = require('react')
const h = require('react-hyperscript')
const withClickOutside = require('react-click-outside')
const TwitterPicker = require('react-color').TwitterPicker
const randomColor = require('randomcolor')
const DocumentTitle = require('react-document-title')
const BodyStyle = require('body-style')

const log = require('./log')
const graphql = require('./graphql')
const mergeColours = require('./helpers').mergeColours
const coloursfragment = require('./helpers').coloursfragment
const onLoggedStateChange = require('./auth').onLoggedStateChange

module.exports = React.createClass({
  getInitialState () {
    return {
      me: null,
      newDomain: ''
    }
  },

  query () {
    graphql.query(`
query {
  me {
    colours { ...${coloursfragment} }
    domains
    plan
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
      h(DocumentTitle, {title: 'User account'}, [
        h(BodyStyle, {style: {backgroundColor}}, [
          h('.tile.is-ancestor', [
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
                        h(Colours, {
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
                        JSON.stringify(this.state.me.meta)
                      ])
                    ])
                  ])
                ])
              ])
            ]),
            h('.tile.is-7', [
              h('.tile.is-child', [
                h('.card.account-plan', [
                  h('.card-header', [
                    h('.card-header-title', 'Plan')
                  ]),
                  h('.card-content', [
                    this.state.me.plan
                  ])
                ])
              ])
            ]),
            h('.tile.is-5')
          ])
        ])
      ])
    )
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
