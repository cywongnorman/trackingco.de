const React = require('react')
const withClickOutside = require('react-click-outside')
const h = require('react-hyperscript')

const graphql = require('./graphql')

module.exports = withClickOutside(React.createClass({
  getInitialState () {
    return {
      name: '',
      writing: false,
      waiting: false,
      error: null
    }
  },

  mutate () {
    graphql.mutate(`
      ($name: String!) {
        createSite(name: $name) {
          code
        }
      }
    `, {name: this.state.name})
    .then(r => {
      this.props.onNewSiteCreated()
      this.setState(this.getInitialState())
    })
    .catch(e => {
      console.log(e.stack)
      this.setState({waiting: false, error: 'failed, please try again.'})
    })
  },

  render () {
    return h('.card.newsite', {id: this.props.code}, [
      h('.card-content', {onClick: this.startWriting}, [
        h('h4.title.is-5', this.state.writing ? 'give a name to the site' : 'add a new site'),
        this.state.writing
        ? h('.card-content', [
          h('form', {onSubmit: this.submit}, [
            h('input.input', {onChange: this.write, value: this.state.name}),
            h('button.button.is-primary', {
              className: this.state.waiting ? 'is-loading' : ''
            }, this.state.error ? this.state.error : 'add')
          ])
        ])
        : h('i.fa.fa-plus')
      ])
    ])
  },

  startWriting () {
    this.setState({writing: true})
  },

  write (e) {
    this.setState({name: e.target.value})
  },

  submit (e) {
    e.preventDefault()
    this.setState({waiting: true})
    this.mutate()
  },

  handleClickOutside () {
    this.setState({writing: false})
  }
}))
