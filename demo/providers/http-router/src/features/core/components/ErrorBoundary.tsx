import React from 'react';
import Container from '@/components/layout/Container';
import {Heading} from '@/components/ui/Heading';
import {Card} from '@/components/ui/Card';

type ErrorBoundaryProps = {
  children: React.ReactNode;
};

class ErrorBoundary extends React.Component<ErrorBoundaryProps> {
  state: {error: Error | undefined; stack: string} = {error: undefined, stack: ''};

  componentDidCatch(error: unknown, info: {componentStack: string}) {
    this.setState({error, stack: info.componentStack});
  }

  render() {
    if (!this.state.error) return this.props.children;

    return (
      <div className="py-8">
        <Container>
          <Card className="bg-red-100 border-red-200 dark:bg-red-950 dark:border-red-900">
            <Heading as="h2" className="text-xl">
              Something went wrong.
            </Heading>
            <p>Please refresh the page and try again.</p>
            <code className="block mt-4">
              <pre>
                {this.state.error.message}
                {this.state.stack}
              </pre>
            </code>
          </Card>
        </Container>
      </div>
    );
  }
}

export {ErrorBoundary};
