const React = require('react')
const h = require('react-hyperscript')
const page = require('page')

const log = require('./log')
const CardsView = require('./CardsView')
const SiteDetail = require('./SiteDetail')
const UserAccount = require('./UserAccount')
const GraphiQL = require('./GraphiQL')

const auth0 = require('./auth').auth0
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
    if (location.hash && location.hash.indexOf('token') !== -1) {
      auth0.parseHash(location.hash, (err, result) => {
        if (err) {
          log.error("error parsing account credentials, you'll be logged out.")
          log.debug(err)
          setToken('')
          return
        }

        log.success("You're now logged in!")
        setToken(result.idToken || result.id_token)
        location.hash = ''
      })
    }

    onLoggedStateChange(isLogged => {
      this.setState({isLogged})
    })

    page('/sites', () =>
      this.setState({route: {component: CardsView}})
    )
    page('/sites/:code', (ctx) =>
      this.setState({route: {component: SiteDetail, props: ctx.params}})
    )
    page('/public/:code', (ctx) =>
      this.setState({route: {component: SiteDetail, props: ctx.params}})
    )
    page('/account', () =>
      this.setState({route: {component: UserAccount}})
    )
    page('/_graphql', () =>
      this.setState({route: {component: GraphiQL}})
    )

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
            h('a.nav-item', 'trackingco.de')
          ]),
          h('.nav-center', [
            this.state.isLogged
            ? h('a', {className: 'nav-item', href: '/account'}, 'account')
            : '',
            this.state.isLogged
            ? h('a', {className: 'nav-item', href: '/sites'}, 'sites')
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
