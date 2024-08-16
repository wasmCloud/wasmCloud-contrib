import Container from '@/components/layout/Container.tsx';
import {Heading} from '@/components/ui/Heading.tsx';
import {ThemeToggle} from '@/features/theme/components/ThemeToggle';
import {useConfig} from '@/services/config/useConfig';

function Navbar() {
  const config = useConfig();

  return (
    <Container className="my-4">
      <div className="flex justify-between">
        <Heading className="text-xl">{config.appName}</Heading>
        <ThemeToggle />
      </div>
    </Container>
  );
}

export {Navbar};
