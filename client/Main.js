const React = require('react')
const h = require('react-hyperscript')
const page = require('page')

const log = require('./log')
const CardsView = require('./CardsView')
const SiteDetail = require('./SiteDetail')
const UserAccount = require('./UserAccount')

const auth0 = require('./auth').auth0
const setToken = require('./auth').setToken
const onLoggedStateChange = require('./auth').onLoggedStateChange

module.exports = React.createClass({
  getInitialState () {
    return {
      isLogged: false,
      email: '',
      route: {
        component: () => h('div'),
        props: {}
      }
    }
  },

  componentDidMount () {
    if (location.hash && location.hash.indexOf('token') !== -1) {
      auth0.parseHash(location.hash, (err, res) => {
        if (err) {
          log.error("error parsing account credentials, you'll be logged out.")
          log.debug(err)
          setToken('')
          return
        }

        log.success("You're now logged in!")
        setToken(res.idToken || res.id_token)
        location.hash = ''
        this.setState({email: res.idTokenPayload['https://trackingco.de/user/email']})
      })
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
    return (
      h('div', [
        h('nav.nav', [
          h('.nav-left', [
            h('a.nav-item', [
              h('img', {src: '/favicon.ico', alt: 'trackingcode logo'})
            ]),
            h('a.nav-item.is-hidden-mobile', 'trackingco.de')
          ]),
          h('.nav-center', [
            this.state.isLogged
            ? h('a.nav-item.is-hidden-touch', this.state.email)
            : '',
            this.state.isLogged
            ? h('a.nav-item', {href: '/account'}, 'account')
            : '',
            this.state.isLogged
            ? h('a.nav-item', {href: '/sites'}, 'sites')
            : h('a.nav-item', {href: auth0.getLoginURL()}, 'login'),
            this.state.isLogged
            ? h('a.nav-item', {key: 'logout', onClick: auth0.logout}, 'logout')
            : h('a.nav-item', {key: 'login', href: auth0.getLoginURL()}, 'start tracking your sites!')
          ])
        ]),
        h(this.state.route.component, this.state.route.props)
      ])
    )
  }
})
