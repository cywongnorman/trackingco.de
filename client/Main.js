const React = require('react')
const h = require('react-hyperscript')
const page = require('page')

const log = require('./log')
const CardsView = require('./CardsView')
const SiteDetail = require('./SiteDetail')
const UserAccount = require('./UserAccount')

const auth = require('./auth').auth
const setToken = require('./auth').setToken
const onLoggedStateChange = require('./auth').onLoggedStateChange

module.exports = React.createClass({
  getInitialState () {
    return {
      isLogged: false,
      route: {
        component: () => h('div'),
        props: {}
      }
    }
  },

  componentDidMount () {
    if (location.search && location.search.indexOf('token') !== -1) {
      let token = auth.tryLogin(location.hash)

      if (!token) {
        log.error("error parsing account credentials, you'll be logged out.")
        setToken('')
        return
      }

      log.success("You're now logged in!")
      setToken(token)
      location.hash = ''
    }

    onLoggedStateChange(isLogged => {
      this.setState({isLogged})
    })

    page('/sites', () => this.setState({route: {component: CardsView}}))
    page('/sites/:code', (ctx) => this.setState({route: {component: SiteDetail, props: ctx.params}}))
    page('/public/:code', (ctx) =>
      this.setState({
        route: {
          component: SiteDetail,
          props: {...ctx.params, public: true}
        }
      })
    )
    page('/account', () => this.setState({route: {component: UserAccount}}))
    page()
  },

  render () {
    let loginURL = `https://accountd.xyz/login-screen?redirect_uri=${location.protocol}//${location.host}/sites&site_name=trackingco.de`

    return (
      h('div', [
        h('nav.nav', [
          h('.nav-left', [
            h('a.nav-item', [
              h('img', {src: '/favicon.ico', alt: 'trackingcode logo'})
            ]),
            h('a.nav-item.is-hidden-mobile', 'trackingco.de')
          ]),
          h('.nav-center', this.state.isLogged
            ? [
              h('a.nav-item.is-hidden-touch', this.state.user),
              h('a.nav-item', {href: '/account'}, 'account'),
              h('a.nav-item', {href: '/sites'}, 'sites'),
              h('a.nav-item', {key: 'logout', onClick: auth.logout}, 'logout')
            ]
            : [
              h('a.nav-item', {href: loginURL}, 'login'),
              h('a.nav-item', {key: 'login', href: loginURL}, 'start tracking your sites!')
            ]
          )
        ]),
        h(this.state.route.component, this.state.route.props)
      ])
    )
  }
})
