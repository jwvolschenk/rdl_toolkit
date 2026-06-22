package rdl

// This file is the single source of truth for the RDL conventions this toolkit
// assumes and enforces. Every read or write goes through here.
//
// Format assumptions
// ------------------
//   • RDL schema version: 2016 only. The toolkit does not understand 2008 R2,
//     2010, 2012, or 2020 builds. Pre-2016 files must be upgraded in SSDT
//     before being handed to these tools.
//
//   • XML encoding: UTF-8 with a leading byte-order mark (BOM: EF BB BF).
//     Visual Studio / SSDT writes RDL files this way; we preserve it.
//     Document.Save enforces BOM presence and CRLF line endings regardless
//     of what the input file had.
//
//   • Line endings: CRLF (Windows-style). LF-only files are normalised on Save.
//
//   • Two XML namespaces appear in RDL 2016:
//       - The default namespace (ReportDefinition 2016) — most elements.
//       - rd: (Report Designer) — vendor-specific elements like rd:ReportID,
//         rd:DataSourceID, rd:SecurityType. XPath in this toolkit matches
//         prefixed names by local-name() because xmlquery's bare XPath
//         requires elements to be in no namespace.
//
//   • Self-closing tags: the XML parser may normalise <Foo /> to <Foo/> on
//     first Save. Both are semantically identical; SSRS reads either.
//     Second Save and beyond is byte-identical (idempotent).

const (
	// RDLNamespace is the default XML namespace on the root <Report> element
	// for RDL 2016 files.
	RDLNamespace = "http://schemas.microsoft.com/sqlserver/reporting/2016/01/reportdefinition"

	// RDNamespace is the vendor namespace used for Report Designer metadata
	// (ReportID, DataSourceID, SecurityType, Generator, etc.).
	RDNamespace = "http://schemas.microsoft.com/SQLServer/reporting/reportdesigner"
)

// bomBytes is the UTF-8 BOM (EF BB BF) prepended on every Save.
var bomBytes = []byte{0xEF, 0xBB, 0xBF}
