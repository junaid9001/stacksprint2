import { ConfigProvider } from '@/src/context/ConfigContext';
import { ToastProvider } from '@/components/ui/ToastContainer';
import { Header } from '@/components/layout/Header';
import { SidebarPreview } from '@/components/layout/SidebarPreview';
import { PresetManager } from '@/components/forms/PresetManager';
import { LanguageArchitectureForm } from '@/components/forms/LanguageArchitectureForm';
import { DatabaseInfraForm } from '@/components/forms/DatabaseInfraForm';
import { OptionalFeaturesForm } from '@/components/forms/OptionalFeaturesForm';
import { FileTogglesForm } from '@/components/forms/FileTogglesForm';
import { RootInitForm } from '@/components/forms/RootInitForm';
import { CustomStructureBuilder } from '@/components/forms/CustomStructureBuilder';

export default function Page() {
  return (
    <ToastProvider>
      <ConfigProvider>
        <main className="app-shell">
          <Header />
          <div className="layout">
            <section className="panel">
              <PresetManager />
              <LanguageArchitectureForm />
              <DatabaseInfraForm />
              <OptionalFeaturesForm />
              <FileTogglesForm />
              <RootInitForm />
              <CustomStructureBuilder />
            </section>

            <SidebarPreview />
          </div>
        </main>
      </ConfigProvider>
    </ToastProvider>
  );
}
