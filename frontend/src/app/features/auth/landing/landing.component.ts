import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './landing.component.html',
})
export class LandingComponent {
  features = [
    {
      icon: '🔗',
      title: 'Any source',
      description: 'URLs, social media, PDFs, chat exports, or raw text — Mirra ingests it all.',
    },
    {
      icon: '🧠',
      title: 'Deep distillation',
      description: 'Extracts tone, humor, values, opinions, and vocabulary into a structured persona.',
    },
    {
      icon: '📤',
      title: 'Export or deploy',
      description: 'Download as a markdown file or deploy as an interactive conversational agent.',
    },
  ];
}
