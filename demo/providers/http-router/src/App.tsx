import Container from '@/components/layout/Container';
import {Navbar} from '@/components/layout/Navbar';
import {Globe} from '@/features/globe/components/Globe';
import {ImageAnalyzer} from '@/features/image-analyzer/components/ImageAnalyzer';
import {Data} from '@/features/statistics/components/Data';
import {useThemeClass} from '@/features/theme/hooks/useThemeClass';

function App() {
  useThemeClass();

  return (
    <>
      <div className="relative flex flex-col justify-start align-middle min-h-full">
        <Navbar />
        <Container>
          <ImageAnalyzer />
        </Container>
        <Globe />
        <Container className="relative z-1">
          <Data />
        </Container>
      </div>
    </>
  );
}

export default App;
