package fastpdf

import (
	"bytes"
	"fmt"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
	"github.com/signintech/gopdf"
	"image"
	"image/jpeg"
	"log"
)

type ItemType byte

const (
	Text ItemType = iota
	BarCode
	QrCode
	Line
	Grid
	Table
)

type FastPdf struct {
	gopdf        *gopdf.GoPdf
	Header       []PdfItem //头部元素
	Body         []PdfItem // 正文元素
	Footer       []PdfItem //底部元素
	TemplatePath string    //底板路径
}

// 新建F
func New(fontPath string, fastPdf FastPdf) FastPdf {
	gp := gopdf.GoPdf{}
	gp.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4}) //595.28, 841.89 = A4
	err := gp.AddTTFFont("font", fontPath)
	if err != nil {
		log.Fatalln(err)
	}
	fastPdf.gopdf = &gp
	fastPdf.newPage() //创建第一页
	return fastPdf
}

// 新建一页
func (pdf *FastPdf) newPage() {
	pdf.gopdf.AddPage()
	if pdf.TemplatePath != "" {
		// Import page 1
		tpl1 := pdf.gopdf.ImportPage(pdf.TemplatePath, pdf.gopdf.GetNumberOfPages(), "/MediaBox")
		// Draw pdf onto page
		pdf.gopdf.UseImportedTemplate(tpl1, 0, 0, 595.28, 841.89)
	}
	pdf.handleItems(pdf.Header) //页头，每页都添加
	pdf.handleItems(pdf.Footer) //页脚，每页都添加
}

// 创建新PDF文件
func (pdf *FastPdf) GetPdf() (fileBytes []byte, err error) {
	pdf.handleItems(pdf.Header)
	pdf.handleItems(pdf.Body)
	pdf.handleItems(pdf.Footer)
	return pdf.gopdf.GetBytesPdf(), nil
}

// 绘制文本
func (pdf *FastPdf) drawText(left, top float64, size int, text string) {
	err := pdf.gopdf.SetFont("font", "", size)
	if err != nil {
		log.Fatalln(err)
	}
	pdf.gopdf.SetX(left)
	pdf.gopdf.SetY(top)
	err = pdf.gopdf.Cell(nil, text)
	if err != nil {
		log.Fatalln(err)
	}
}

func (pdf *FastPdf) drawBarCode(left, top float64, width, height int, text string) {
	barCodeImage := strToBarCode(text, width, height)
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, barCodeImage, &jpeg.Options{Quality: 100})
	imgH1, err := gopdf.ImageHolderByBytes(buf.Bytes())
	if err != nil {
		log.Fatalln(err)
	}
	pdf.gopdf.ImageByHolder(imgH1, left, top, nil)
}

func (pdf *FastPdf) drawLine(x1, y1, x2, y2 float64) {
	pdf.gopdf.SetLineWidth(1)
	pdf.gopdf.SetLineType("")
	pdf.gopdf.Line(x1, y1, x2, y2)
}

func (pdf *FastPdf) drawTable(left, top, width, height float64, row, column int) {
	pdf.gopdf.SetLineWidth(1)
	pdf.gopdf.SetLineType("")
	distanceH := width / float64(row)     //横线之间的间距
	distanceW := height / float64(column) //竖线之间的间距
	//Pen lineColor = new Pen(p.PrtColor, 0.2f);
	for i := 0; i < row+1; i++ {
		//画横线
		y := top + distanceH*float64(i)
		pdf.gopdf.Line(left, y, left+width, y)
	}
	for i := 0; i < column+1; i++ {
		//画竖线
		x := left + distanceW*float64(i)
		pdf.gopdf.Line(x, top, x, top+float64(height))
	}
}

func (pdf *FastPdf) drawQrCode(left, top float64, size int, text string) {
	qrCode := strToQrCode(text, size)
	//保存到新文件中
	//newfile, _ := os.Create("imgs/qr.png")
	//err = jpeg.Encode(newfile, qrCode, &jpeg.Options{Quality: 100})
	//if err != nil {
	//	fmt.Println(err)
	//}
	//pdf.Image("imgs/qr.png", item.Start[0], item.Start[1], nil) //print image

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, qrCode, &jpeg.Options{Quality: 100})
	imgH1, err := gopdf.ImageHolderByBytes(buf.Bytes())
	if err != nil {
		log.Fatalln(err)
	}
	pdf.gopdf.ImageByHolder(imgH1, left, top, nil)
}

func (pdf *FastPdf) handleItems(items []PdfItem) {
	for _, item := range items {
		switch item.Type {
		case Text:
			{
				pdf.drawText(item.Left, item.Top, item.Size, item.Text)
			}
			break
		case BarCode:
			{
				pdf.drawBarCode(item.Left, item.Top, item.Width, item.Height, item.Text)
			}
			break
		case QrCode:
			{
				pdf.drawQrCode(item.Left, item.Top, item.Size, item.Text)
			}
			break
		case Grid:
			{
				pdf.drawTable(item.Left, item.Top, float64(item.Width), float64(item.Height), item.Row, item.Column)
			}
			break
		case Table:
			{
				if len(item.TableColumn) == 0 {
					break
				}
				//先画标题框
				left := item.Left
				top := item.Top
				nextTop := top + float64(item.Height)
				//绘制标题栏
				pdf.drawLine(left, top, left, nextTop) //先画左边框 仅一行
				for _, column := range item.TableColumn {
					nextLeft := left + float64(column.Width)
					pdf.drawLine(left, top, nextLeft, top)                                   //顶部边框
					pdf.drawLine(left, nextTop, nextLeft, nextTop)                           //底部边框
					pdf.drawLine(nextLeft, top, nextLeft, nextTop)                           //画右边框
					pdf.drawText(left+2, top+2, int(float64(item.Height)*0.8), column.Title) //写内容
					left = nextLeft
				}
				top = nextTop //下移一行
				nextTop = top + float64(item.Height)
				left = item.Left //左边复位
				//再绘制数据行

				for _, row := range item.TableData {
					pdf.drawLine(left, top, left, nextTop) //先画左边框 仅一行
					for cellIndex := 0; cellIndex < len(item.TableColumn); cellIndex++ {
						nextLeft := left + float64(item.TableColumn[cellIndex].Width)
						pdf.drawLine(left, nextTop, nextLeft, nextTop)                                                   //底部边框
						pdf.drawLine(nextLeft, top, nextLeft, nextTop)                                                   //画右边框
						pdf.drawText(left+2, top+2, int(float64(item.Height)*0.8), row[item.TableColumn[cellIndex].Key]) //写内容
						left = nextLeft
					}
					top = nextTop //下移一行
					nextTop = top + float64(item.Height)
					left = item.Left //左边复位
				}
			}
		}
	}
}

// 生成二维码
func strToQrCode(content string, size int) image.Image {
	qrCode, _ := qr.Encode(content, qr.M, qr.Auto)
	// 设置图片像素大小
	qrCode, _ = barcode.Scale(qrCode, size, size)
	return qrCode
}

// 生成条码
func strToBarCode(content string, width int, height int) image.Image {
	barCodeCs, err := code128.Encode(content)
	if err != nil {
		fmt.Println(err)
	}
	// 设置图片像素大小
	barCode, err := barcode.Scale(barCodeCs, width, height)
	if err != nil {
		fmt.Println(err)
	}
	return barCode
}

type PdfItem struct {
	Type   ItemType `json:"type"`
	Left   float64  `json:"left"`
	Top    float64  `json:"top"`
	Size   int      `json:"size"`   //	大小
	Width  int      `json:"width"`  // 宽
	Height int      `json:"height"` // 高 在table里指的是行高，不指定总高
	//FontStyle string `json:"fontStyle"` //	字体样式 默认：Arial
	Text        string              `json:"text"`   // 文本内容
	Column      int                 `json:"column"` // 列数
	Row         int                 `json:"row"`    // 行数
	TableColumn []TableColumn       `json:"tableColumn"`
	TableData   []map[string]string `json:"tableData"` //行 列
	//PageSize    int           `json:"pageSize"`
}

type TableColumn struct {
	Width int    `json:"width"` // 列宽
	Title string `json:"title"` // 标题内容
	Key   string `json:"key"`   // 数据key
}
