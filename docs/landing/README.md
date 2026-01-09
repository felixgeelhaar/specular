# Specular Landing Page

A distinctive, production-grade landing page for Specular CLI.

## Design

**Aesthetic**: Mission Control / Command Center
- Conveys governance, precision, and enterprise-grade oversight
- Deep charcoal base with electric cyan accents
- Blueprint-style grid background
- Terminal-inspired UI elements with animations

**Typography**:
- **Syne** - Geometric display font for bold headlines
- **IBM Plex Mono** - Monospace for code and technical elements
- **IBM Plex Sans** - Clean body text

**Key Sections**:
1. Hero with animated terminal preview
2. Workflow pipeline visualization
3. Feature cards with accent colors
4. Installation methods
5. Stats showcase
6. Call-to-action

## Deployment

### GitHub Pages

1. **Via GitHub Settings**:
   - Go to Repository → Settings → Pages
   - Source: Deploy from branch
   - Branch: `main` → `/docs/landing`
   - Save

2. **Via GitHub Actions** (recommended):
   ```yaml
   name: Deploy Landing Page
   on:
     push:
       branches: [main]
       paths: ['docs/landing/**']

   jobs:
     deploy:
       runs-on: ubuntu-latest
       permissions:
         pages: write
         id-token: write
       steps:
         - uses: actions/checkout@v4
         - uses: actions/configure-pages@v4
         - uses: actions/upload-pages-artifact@v3
           with:
             path: docs/landing
         - uses: actions/deploy-pages@v4
   ```

### Local Preview

```bash
# Using Python
cd docs/landing
python -m http.server 8000
# Open http://localhost:8000

# Using Node.js
npx serve docs/landing
```

## Customization

### Colors

Edit CSS variables in `index.html`:

```css
:root {
  --accent-cyan: #00d4ff;     /* Primary accent */
  --accent-amber: #f59e0b;    /* Warning/governance */
  --accent-emerald: #10b981;  /* Success states */
  --accent-violet: #8b5cf6;   /* Secondary accent */
  --bg-void: #040608;         /* Darkest background */
  --bg-primary: #0a0f14;      /* Main background */
}
```

### Content

Update sections directly in `index.html`:
- Version badge in hero section
- Terminal preview commands
- Feature card descriptions
- Stats values
- Installation commands

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Performance

- Single HTML file (~30KB)
- No external JavaScript dependencies
- CSS animations with GPU acceleration
- Google Fonts loaded asynchronously
