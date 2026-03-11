# ADR-002: Form Framework Selection

**Status:** Approved  
**Date:** 2026-03-09  
**Decision Maker:** Architect  
**Implementation Features:** #4 (Silences), #30-#32 (Config Builder)

---

## Context

Phase 1 Visualization requires multiple complex forms:

**Feature #4 (Silences):**
- Duration picker (pre-defined + custom date/time)
- Matcher editor (label=value pairs, add/remove rows)
- Comment field

**Feature #5 (Config Builder) — Complex Forms:**
- RouteBuilder (deeply nested forms, add/remove child routes)
- ReceiverForm (type-selector + type-specific fields, 8+ receiver types)
- TimeIntervalEditor (date/time, weekday/month multiselect)
- ConfigReview (multi-step form flow)

**Requirements:**
- Handle validation (matchers syntax, dates, required fields)
- Support dynamic fields (add/remove rows)
- Type-safe (TypeScript support)
- Good UX (error messages, focus management)
- Lightweight (prefer < 50KB additional dependency)

**Constraints:**
- Must work well in Svelte/SvelteKit
- Must be maintained/actively developed
- Should support async validation (for matcher syntax check)

---

## Options Considered

### Option A: Custom Svelte Forms Pattern (Recommended)

**Description:**  
Use vanilla Svelte + Zod validation library. Build form components (form.svelte, FormGroup.svelte) as reusable wrappers. No external form library.

**Pattern:**
```svelte
<script>
  import { z } from 'zod';
  
  const schema = z.object({
    matchers: z.array(z.string()).refine(...)
  });
  
  let formData = {};
  let errors = {};

  function validate() {
    const result = schema.safeParse(formData);
    if (!result.success) {
      errors = result.error.flatten().fieldErrors;
    }
    return result.success;
  }

  function submit() {
    if (validate()) {
      // submit formData
    }
  }
</script>

<form on:submit|preventDefault={submit}>
  <FormGroup>
    <MatcherEditor bind:value={formData.matchers} {errors} />
  </FormGroup>
  <button>Save</button>
</form>
```

**Pros:**
- ✅ Zero dependencies (except Zod for validation)
- ✅ Full control over behavior & styling
- ✅ Lightweight (Zod is ~40KB gzipped)
- ✅ Svelte-native (reactive, simple)
- ✅ Easy to debug
- ✅ No learning curve (team knows Svelte)

**Cons:**
- ❌ Manual state management (more boilerplate)
- ❌ Field-level updates must be managed per form
- ❌ No built-in async validation middleware
- ❌ Must handle focus/tab order manually

**Estimated Effort:**
- Setup (base components): 1 day
- Feature #4 (Silences): 1.5 days
- Feature #5 (Config): 3 days
- Total: 5.5 days

**Cost:** Free (Zod is MIT)

---

### Option B: Svelte Forms Library (@sveltejs/form)

**Description:**  
Official Svelte form handling package (lightweight, framework-aligned).

**Note:** As of 2026, this is still in development/proposal phase. Check current status.

**Pros:**
- ✅ Tight Svelte integration
- ✅ Lightweight and opinionated
- ✅ Likely good TypeScript support

**Cons:**
- ⚠️ Unstable/young ecosystem
- ⚠️ May not be production-ready
- ❌ Limited documentation (if still developing)
- ❌ Small community
- ❌ Validation library still needs to be chosen

**Estimated Effort:**
- Learning: 2-3 days (API still stabilizing)
- Feature #4: 1.5 days
- Feature #5: 3 days
- Total: 6.5-7.5 days

**Risk:** High (API might change mid-feature)

---

### Option C: FormKit (Full-Featured)

**Description:**  
Comprehensive form framework with validation, error handling, accessibility built-in.

**Pros:**
- ✅ Full-featured (validation, async, accessibility)
- ✅ Great DX (automatic form state management)
- ✅ Handles complexity (nested forms, dynamic fields)
- ✅ Good documentation
- ✅ Active community

**Cons:**
- ❌ Heavy (~200KB+ gzipped, large)
- ❌ Opinionated styling (customization required for design system)
- ❌ Svelte support good but not primary (Vue is primary)
- ❌ Learning curve (new API)
- ❌ Overkill for simple forms

**Estimated Effort:**
- Learning: 2 days
- Feature #4: 1 day
- Feature #5: 2.5 days
- Total: 5.5 days

**Cost:** Free community, paid enterprise version

---

### Option D: Formik (Established Standard)

**Description:**  
Industry-standard form library (originally React, now multi-framework).

**Pros:**
- ✅ Very well-documented
- ✅ Large community
- ✅ Proven in production

**Cons:**
- ❌ Heavy (~60KB+ gzipped)
- ❌ Designed for React (Svelte integration awkward)
- ❌ Learning curve
- ❌ Not Svelte-first

**Estimated Effort:**
- Learning: 2-3 days
- Integration: higher due to React-first design
- Total: 6+ days

**Risk:** Not ideal for Svelte

---

## Decision

**✅ APPROVED: Custom Svelte Forms + Zod (Option A)**

**Rationale:**

1. **Lightweight & Fast:**
   - No heavy dependencies
   - Small bundle impact (Zod ~40KB)
   - Fast form renders & updates

2. **Svelte-Native:**
   - Aligns with Svelte's reactive model
   - No abstraction layers
   - Easy to understand for team
   - No impedance mismatch

3. **Full Control:**
   - Custom error messages (important for matcher syntax hints)
   - Full styling control (matches design system)
   - Can implement advanced features (async validation, conditional fields)

4. **Type Safety:**
   - Zod provides compile-time + runtime type validation
   - Works seamlessly with TypeScript
   - Great error messages for debugging

5. **Time Efficiency:**
   - Team already knows Svelte
   - No new framework to learn
   - Direct implementation without learning curve

6. **Maintainability:**
   - Code is explicit & debuggable
   - No magic, easy to modify
   - Team can fix bugs immediately

---

## Implementation Details

### Core Form Components

**File:** `web/src/components/Form/`

```
Form/
├── FormGroup.svelte       # Wrapper for label + error + field
├── FormSubmit.svelte      # Submit button with loading state
├── FormError.svelte       # Error message display
├── TextInput.svelte       # Text field with validation
├── DateInput.svelte       # Date picker integration
├── MultiSelect.svelte     # Multi-choice selector
├── DynamicFieldArray.svelte # Add/remove row functionality
└── FormContext.svelte     # Provides form state (via context)
```

### Validation Library: Zod

**Installation:**
```bash
npm install zod
```

### Example: SilenceForm Validation

**File:** `web/src/routes/silences/SilenceForm.svelte`

```svelte
<script>
  import { z } from 'zod';
  import FormGroup from '../../components/Form/FormGroup.svelte';
  import FormSubmit from '../../components/Form/FormSubmit.svelte';
  import MatcherEditor from '../../components/MatcherEditor.svelte';
  import DurationPicker from '../../components/DurationPicker.svelte';

  const silenceSchema = z.object({
    matchers: z.array(
      z.object({
        label: z.string().min(1, "Label required"),
        value: z.string().min(1, "Value required"),
        isRegex: z.boolean()
      })
    ).min(1, "At least one matcher required"),
    duration: z.number().min(300, "Minimum 5 minutes"),
    comment: z.string().optional(),
    createdBy: z.string().optional()
  });

  let formData = {
    matchers: [],
    duration: 3600, // 1 hour default
    comment: '',
    createdBy: ''
  };
  
  let errors = {};

  async function validate() {
    const result = await silenceSchema.safeParseAsync(formData);
    if (!result.success) {
      errors = result.error.flatten().fieldErrors;
      return false;
    }
    return true;
  }

  async function submit() {
    if (await validate()) {
      const response = await fetch('/api/silences', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData)
      });
      // Handle response
    }
  }
</script>

<form on:submit|preventDefault={submit}>
  <FormGroup label="Matchers" error={errors.matchers?.[0]}>
    <MatcherEditor bind:matchers={formData.matchers} />
  </FormGroup>

  <FormGroup label="Duration" error={errors.duration?.[0]}>
    <DurationPicker bind:duration={formData.duration} />
  </FormGroup>

  <FormGroup label="Comment" error={errors.comment?.[0]}>
    <textarea bind:value={formData.comment} />
  </FormGroup>

  <FormSubmit loading={isSubmitting}>Save Silence</FormSubmit>
</form>
```

### Async Validation Example (for matcher syntax)

```typescript
const matcherSchema = z.object({
  matchers: z.array(z.string())
}).refine(
  async (data) => {
    // Server-side validation of matcher syntax
    const result = await fetch('/api/validate-matchers', {
      method: 'POST',
      body: JSON.stringify(data)
    });
    return result.ok;
  },
  { message: "Invalid matcher syntax" }
);
```

---

## Component Library Structure

### FormGroup (Reusable Wrapper)

```svelte
<script>
  export let label = '';
  export let error = '';
  export let required = false;
  export let helpText = '';
</script>

<div class="form-group">
  {#if label}
    <label>{label} {#if required}<span class="required">*</span>{/if}</label>
  {/if}
  <slot />
  {#if error}
    <div class="error">{error}</div>
  {/if}
  {#if helpText}
    <div class="help-text">{helpText}</div>
  {/if}
</div>

<style>
  .form-group { margin-bottom: 1rem; }
  label { font-weight: bold; display: block; }
  .error { color: #dc2626; font-size: 0.875rem; margin-top: 0.25rem; }
  .help-text { color: #6b7280; font-size: 0.875rem; margin-top: 0.25rem; }
</style>
```

### DynamicFieldArray (Add/Remove Rows)

```svelte
<script>
  export let fields = [];
  export let label = 'Add';
  export let error = '';

  function addField() {
    fields = [...fields, {}];
  }

  function removeField(index) {
    fields = fields.filter((_, i) => i !== index);
  }
</script>

<div class="field-array">
  {#each fields as field, i (i)}
    <div class="row">
      <slot {field} {i} />
      <button on:click={() => removeField(i)} type="button">Remove</button>
    </div>
  {/each}
  <button on:click={addField} type="button">+ {label}</button>
</div>
```

---

## Testing Strategy

### Unit Tests (Vitest)

```typescript
import { z } from 'zod';
import { silenceSchema } from './silenceForm';

describe('SilenceForm validation', () => {
  it('should validate valid silence data', async () => {
    const valid = {
      matchers: [{ label: 'severity', value: 'critical' }],
      duration: 3600,
      comment: 'Maintenance'
    };
    const result = await silenceSchema.safeParseAsync(valid);
    expect(result.success).toBe(true);
  });

  it('should reject missing matchers', async () => {
    const invalid = {
      matchers: [],
      duration: 3600
    };
    const result = await silenceSchema.safeParseAsync(invalid);
    expect(result.success).toBe(false);
  });
});
```

### Component Tests (Svelte Testing Library)

```typescript
import { render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import SilenceForm from './SilenceForm.svelte';

describe('SilenceForm component', () => {
  it('should submit valid form', async () => {
    render(SilenceForm);
    const input = screen.getByLabelText('Matchers');
    await userEvent.type(input, 'severity=critical');
    
    const button = screen.getByText('Save Silence');
    await userEvent.click(button);
    
    // Assert submission happened
  });
});
```

---

## Migration Path (If Needed)

If custom forms become too cumbersome (unlikely):

1. **Step 1:** Replace Zod with FormKit
2. **Step 2:** Keep Form* components, map to FormKit APIs
3. **Effort:** 1-2 days

---

## Dependencies & Coordination

- **Depends on:** Zod (validation library only)
- **Used by:** Feature #4 (Silences), Feature #5 (Config)
- **Blocks:** None (no other features depend on form framework)

---

## Success Criteria

- [ ] FormGroup component reusable across all forms
- [ ] Zod validation works for complex schemas
- [ ] Error messages display inline with field
- [ ] DynamicFieldArray (add/remove rows) functional
- [ ] Async validation works (matcher syntax check)
- [ ] Unit tests pass (≥80% coverage)
- [ ] Component tests pass
- [ ] Multi-step forms work (ConfigReview)
- [ ] Nested object validation works (routes in Config)

---

## Timeline

- **Duration:** Concurrent with Feature #4 & #5
  - Day 1: Set up Form* components & Zod schema
  - Days 2-5: Feature #4 (Silences) implementation
  - Days 6-10: Feature #5 (Config) implementation

---

## Related ADRs

- ADR-001: Routing Tree Visualization
- ADR-003: Config Storage & Rollback Strategy
- ADR-004: Real-time Update Strategy

---

## Approval Sign-off

- **Architect:** ✅ Approved 2026-03-09
- **Developer:** ⬜ To confirm on implementation
- **Security:** ✅ No security concerns (validation prevents injection)

---

## Notes

1. **Zod is lightweight:** Small dependency, easy to add/remove if needed
2. **Svelte's reactivity shines:** Manual state management works naturally with Svelte's stores
3. **Custom components are reusable:** FormGroup, DynamicFieldArray patterns apply across all features
4. **Async validation:** Important for matcher syntax checking — Zod supports it naturally

---

**End of ADR-002**
