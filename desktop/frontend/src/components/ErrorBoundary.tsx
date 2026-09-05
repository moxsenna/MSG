import * as React from 'react';
import { MessageBar, MessageBarBody, Button } from '@fluentui/react-components';
type S={hasError:boolean;error?:Error};
export default class ErrorBoundary extends React.Component<React.PropsWithChildren,S>{state:S={hasError:false};static getDerivedStateFromError(e:Error){return{hasError:true,error:e}};componentDidCatch(e:Error){console.error(e)};render(){if(this.state.hasError)return(<div style={{padding:16}}><MessageBar intent='error'><MessageBarBody>Something went wrong: {this.state.error?.message}</MessageBarBody></MessageBar><Button onClick={()=>this.setState({hasError:false})}>Dismiss</Button></div>);return this.props.children;}}
