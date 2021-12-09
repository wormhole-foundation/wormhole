# Design Document Template

<i>
This is a design document template, used to help project members discuss and understand design decisions that you made, especially during code review. It is also useful as a reference in the future, even for yourself.

Creating a design document is highly recommended for any programming work that **takes more than two calendar days of wall clock time to implement** - any time spent on design discussions here is time saved from having to rewrite code based on wrong assumptions.

INSTRUCTIONS: Fill out this template by making a copy of this file and removing/filling out the italicized sections. All sections are optional and can be left out when not applicable - but try to fill as many as you can.

We use soft wraps - rather than hard wraps at a specific line length. This makes it easier to copy, edit and review prose. We recommend configuring your editor to use soft wraps for Markdown files.
</i>


## Objective

<i>

One or two sentence descriptions of what problem you are trying to solve.

</i>


## Background

<i>

Describe any systems, components, and/or technologies which are required for but are not part of this design proposal. Assume your audience is unfamiliar with the design space. Only provide as much detail here as necessary for a reader to make sense of your design proposal and know where to look if they want additional detail. Include links to relevant references both internal and external.

This is also the place to discuss the limitations of prior designs and implementations. Wait to describe your solutions to those limitations as part of the Proposal section.

</i>


## Goals

<i>

List of things your design explicitly tries to accomplish. The design must meet all of these to be acceptable.

- ...
- ...

</i>
  

## Non-Goals

<i>

List of things your design explicitly chooses to not address.

- A design document is not an analysis, business plan, specification or manual.
- ...

</i>


## Overview

<i>

If ‘Detailed Design’ underneath spans more than a few paragraphs, provide an overview of the design here. This is a great place to provide some high-level diagrams showing your design.

</i>


## Detailed Design

<i>

Describe your design in terms of what you covered in the Background section. Provide an overview of the major components in your solution, how users and external components will interact with them, and how the components interact with each other internally.

This section should contain specific details of what you plan to build. Cite specific design patterns, technologies, and components that are required to build the design. Avoid specifying ancillary details such as specific libraries, programming languages, etc unless their use is explicitly part of the design. For example, prefer to specify “a SQL server” rather than “MySQL” unless your design depends on features that are only available with MySQL or MySQL was specifically chosen for this design after an evaluation and team discussion.

</i>


### API / database schema

<i>

Any interface definition specified (ie. protobuf files, database schemas) should be explicitly cited here.

</i>


## Caveats

<i>

Any parts of your proposed design that are tricky to implement or which you think might be problematic to implement in practice should be listed here. It’s better to admit to being unsure about something than to try to pretend you know how to solve everything up front. Your reviewers might be able to help, so also consider explaining your existing research into solving these caveats.

</i>


## Alternatives Considered

<i>

For each major design choice in your proposal, add a subsection here to describe each alternative you considered and why you rejected it. A reader should be able to understand your reasoning that led to your proposal even if they disagree.

</i>

## Security Considerations

<i>

This is the place to mention how your design approaches security: what surfaces does it exposed to (un)trusted users, what (un)trusted data it processes, what privileges will it run with in production.

</i>
